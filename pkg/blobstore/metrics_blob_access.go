package blobstore

import (
	"context"
	"io"
	"math"
	"time"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	blobAccessOperationsStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "blob_access_operations_started_total",
			Help:      "Total number of operations started on blob access objects.",
		},
		[]string{"name", "operation"})
	blobAccessOperationsDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "blob_access_operations_duration_seconds",
			Help:      "Amount of time spent per operation on blob access objects, in seconds.",
			Buckets:   prometheus.ExponentialBuckets(0.001, math.Pow(10.0, 1.0/3.0), 6*3+1),
		},
		[]string{"name", "operation"})
)

func init() {
	prometheus.MustRegister(blobAccessOperationsStartedTotal)
	prometheus.MustRegister(blobAccessOperationsDurationSeconds)
}

type metricsBlobAccess struct {
	blobAccess BlobAccess
	name       string
}

// NewMetricsBlobAccess creates an adapter for BlobAccess that adds
// basic instrumentation in the form of Prometheus metrics.
func NewMetricsBlobAccess(blobAccess BlobAccess, name string) BlobAccess {
	return &metricsBlobAccess{
		blobAccess: blobAccess,
		name:       name,
	}
}

func (ba *metricsBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "Get").Inc()
	timeStart := time.Now()
	r := ba.blobAccess.Get(ctx, instance, digest)
	blobAccessOperationsDurationSeconds.WithLabelValues(ba.name, "Get").Observe(time.Now().Sub(timeStart).Seconds())
	return r
}

func (ba *metricsBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "Put").Inc()
	timeStart := time.Now()
	err := ba.blobAccess.Put(ctx, instance, digest, sizeBytes, r)
	blobAccessOperationsDurationSeconds.WithLabelValues(ba.name, "Put").Observe(time.Now().Sub(timeStart).Seconds())
	return err
}

func (ba *metricsBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	blobAccessOperationsStartedTotal.WithLabelValues(ba.name, "FindMissing").Inc()
	timeStart := time.Now()
	digests, err := ba.blobAccess.FindMissing(ctx, instance, digests)
	blobAccessOperationsDurationSeconds.WithLabelValues(ba.name, "FindMissing").Observe(time.Now().Sub(timeStart).Seconds())
	return digests, err
}
