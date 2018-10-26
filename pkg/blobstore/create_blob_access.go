package blobstore

import (
	"errors"
	"fmt"
	"io/ioutil"

	pb "github.com/EdSchouten/bazel-buildbarn/pkg/proto/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/proto"
)

// CreateBlobAccessObjectsFromConfig creates a pair of BlobAccess
// objects for the Content Addressable Storage and Action cache based on
// a configuration file.
func CreateBlobAccessObjectsFromConfig(configurationFile string) (BlobAccess, BlobAccess, error) {
	data, err := ioutil.ReadFile(configurationFile)
	if err != nil {
		return nil, nil, err
	}
	var config pb.BlobstoreConfiguration
	if err := proto.UnmarshalText(string(data), &config); err != nil {
		return nil, nil, err
	}
	contentAddressableStorage, err := createBlobAccess(config.ContentAddressableStorage, "cas", util.KeyDigestWithoutInstance)
	if err != nil {
		return nil, nil, err
	}
	actionCache, err := createBlobAccess(config.ActionCache, "ac", util.KeyDigestWithInstance)
	if err != nil {
		return nil, nil, err
	}
	return NewMetricsBlobAccess(
		NewMerkleBlobAccess(NewEmptyBlobFilteringBlobAccess(contentAddressableStorage)),
		"cas_merkle"), actionCache, nil
}

func createBlobAccess(config *pb.BlobAccessConfiguration, storageType string, digestKeyer util.DigestKeyer) (BlobAccess, error) {
	var implementation BlobAccess
	var backendType string
	switch backend := config.Backend.(type) {
	case *pb.BlobAccessConfiguration_Redis:
		backendType = "redis"
		implementation = NewRedisBlobAccess(
			redis.NewClient(
				&redis.Options{
					Addr: backend.Redis.Endpoint,
					DB:   int(backend.Redis.Db),
				}),
			digestKeyer)
	case *pb.BlobAccessConfiguration_Remote:
		backendType = "remote"
		implementation = NewRemoteBlobAccess(backend.Remote.Address, storageType)
	case *pb.BlobAccessConfiguration_S3:
		backendType = "s3"
		session := session.New(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(backend.S3.AccessKeyId, backend.S3.SecretAccessKey, ""),
			Endpoint:         &backend.S3.Endpoint,
			Region:           &backend.S3.Region,
			DisableSSL:       &backend.S3.DisableSsl,
			S3ForcePathStyle: aws.Bool(true),
		})
		s3 := s3.New(session)
		// Set the uploader concurrency to 1 to drastically reduce memory usage.
		// TODO(edsch): Maybe the concurrency can be left alone for this process?
		uploader := s3manager.NewUploader(session)
		uploader.Concurrency = 1
		implementation = NewS3BlobAccess(
			s3,
			uploader,
			&backend.S3.Bucket,
			digestKeyer)
	case *pb.BlobAccessConfiguration_SizeDistinguishing:
		backendType = "size_distinguishing"
		small, err := createBlobAccess(backend.SizeDistinguishing.Small, storageType, digestKeyer)
		if err != nil {
			return nil, err
		}
		large, err := createBlobAccess(backend.SizeDistinguishing.Large, storageType, digestKeyer)
		if err != nil {
			return nil, err
		}
		implementation = NewSizeDistinguishingBlobAccess(small, large, backend.SizeDistinguishing.CutoffSizeBytes)
	default:
		return nil, errors.New("Configuration did not contain a backend")
	}
	return NewMetricsBlobAccess(implementation, fmt.Sprintf("%s_%s", storageType, backendType)), nil
}
