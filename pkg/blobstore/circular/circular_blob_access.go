package circular

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OffsetStore maps a digest to an offset within the data file. This is
// where the blob's contents may be found.
type OffsetStore interface {
	Get(digest SimpleDigest, cursors Cursors) (uint64, int64, bool, error)
	Put(digest SimpleDigest, offset uint64, length int64, cursors Cursors) error
}

// DataStore is where the data corresponding with a blob is stored. Data
// can be accessed by providing an offset within the data store and its
// length.
type DataStore interface {
	Put(r io.Reader, offset uint64) error
	Get(offset uint64, size int64) io.ReadCloser
}

// StateStore is where global metadata of the circular storage backend
// is stored, namely the read/write cursors where data is currently
// being stored in the data file.
type StateStore interface {
	Get() (Cursors, error)
	Put(cursors Cursors) error
}

type circularBlobAccess struct {
	// Fields that are constant or lockless.
	dataStore DataStore
	dataSize  uint64

	// Fields protected by the lock.
	lock        sync.Mutex
	cursors     Cursors
	offsetStore OffsetStore
	stateStore  StateStore
}

// NewCircularBlobAccess creates a new circular storage backend. Instead
// of writing data to storage directly, all three storage files are
// injected through separate interfaces.
func NewCircularBlobAccess(offsetStore OffsetStore, dataStore DataStore, dataSize uint64, stateStore StateStore) (blobstore.BlobAccess, error) {
	cursors, err := stateStore.Get()
	if err != nil {
		return nil, err
	}

	ba := &circularBlobAccess{
		offsetStore: offsetStore,
		dataStore:   dataStore,
		dataSize:    dataSize,
		stateStore:  stateStore,
		cursors:     cursors,
	}
	go ba.flushStateStore()
	return ba, nil
}

func (ba *circularBlobAccess) flushStateStore() {
	for {
		time.Sleep(time.Minute)

		ba.lock.Lock()
		cursors := ba.cursors
		ba.lock.Unlock()

		if err := ba.stateStore.Put(cursors); err != nil {
			log.Print("Failed to write to state store: ", err)
		}
	}
}

func (ba *circularBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	ba.lock.Lock()
	offset, length, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.cursors)
	ba.lock.Unlock()
	if err != nil {
		return 0, nil, err
	} else if ok {
		return length, ba.dataStore.Get(offset, length), nil
	}
	return 0, nil, status.Errorf(codes.NotFound, "Blob not found")
}

func (ba *circularBlobAccess) Put(ctx context.Context, digest *util.Digest, sizeBytes int64, r io.ReadCloser) error {
	defer r.Close()

	// Allocate space in the data store.
	ba.lock.Lock()
	offset := ba.cursors.Allocate(sizeBytes, ba.dataSize)
	ba.lock.Unlock()

	// Write the data to storage.
	if err := ba.dataStore.Put(r, offset); err != nil {
		return err
	}

	var err error
	ba.lock.Lock()
	if ba.cursors.Contains(offset, sizeBytes) {
		err = ba.offsetStore.Put(NewSimpleDigest(digest), offset, sizeBytes, ba.cursors)
	} else {
		err = errors.New("Data became stale before write completed")
	}
	ba.lock.Unlock()
	return err

}

func (ba *circularBlobAccess) Delete(ctx context.Context, digest *util.Digest) error {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	if offset, length, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.cursors); err != nil {
		return err
	} else if ok {
		ba.cursors.Invalidate(offset, length)
	}
	return nil
}

func (ba *circularBlobAccess) FindMissing(ctx context.Context, digests []*util.Digest) ([]*util.Digest, error) {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	var missingDigests []*util.Digest
	for _, digest := range digests {
		if _, _, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.cursors); err != nil {
			return nil, err
		} else if !ok {
			missingDigests = append(missingDigests, digest)
		}
	}
	return missingDigests, nil
}
