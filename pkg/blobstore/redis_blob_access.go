package blobstore

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/go-redis/redis"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type redisBlobAccess struct {
	redisClient *redis.Client
	blobKeyer   util.DigestKeyer
}

func NewRedisBlobAccess(redisClient *redis.Client, blobKeyer util.DigestKeyer) BlobAccess {
	return &redisBlobAccess{
		redisClient: redisClient,
		blobKeyer:   blobKeyer,
	}
}

func (ba *redisBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	if err := ctx.Err(); err != nil {
		return &errorReader{err: err}
	}
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return &errorReader{err: err}
	}
	value, err := ba.redisClient.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return &errorReader{err: status.Errorf(codes.NotFound, err.Error())}
		}
		return &errorReader{err: err}
	}
	return ioutil.NopCloser(bytes.NewBuffer(value))
}

func (ba *redisBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	value, err := ioutil.ReadAll(r)
	r.Close()
	if err != nil {
		return err
	}
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	return ba.redisClient.Set(key, value, 0).Err()
}

func (ba *redisBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(digests) == 0 {
		return nil, nil
	}

	// Execute "EXISTS" requests all in a single pipeline.
	pipeline := ba.redisClient.Pipeline()
	var cmds []*redis.IntCmd
	for _, digest := range digests {
		key, err := ba.blobKeyer(instance, digest)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, pipeline.Exists(key))
	}
	_, err := pipeline.Exec()
	if err != nil {
		return nil, err
	}

	var missing []*remoteexecution.Digest
	for i, cmd := range cmds {
		if cmd.Val() == 0 {
			missing = append(missing, digests[i])
		}
	}
	return missing, nil
}
