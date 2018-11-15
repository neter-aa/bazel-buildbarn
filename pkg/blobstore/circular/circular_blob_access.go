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

type OffsetStore interface {
	Get(digest SimpleDigest, minOffset uint64, maxOffset uint64) (uint64, int64, bool, error)
	Put(digest SimpleDigest, offset uint64, length int64, minOffset uint64, maxOffset uint64) error
}

type DataStore interface {
	Put(r io.Reader, offset uint64) error
	Get(offset uint64, size int64) io.ReadCloser
}

type StateStore interface {
	Get() (uint64, uint64, error)
	Put(readCursor uint64, writeCursor uint64) error
}

type circularBlobAccess struct {
	// Fields that are constant or lockless.
	dataStore DataStore
	dataSize  uint64

	// Fields protected by the lock.
	lock        sync.Mutex
	readCursor  uint64
	writeCursor uint64
	offsetStore OffsetStore
	stateStore  StateStore
}

func NewCircularBlobAccess(offsetStore OffsetStore, dataStore DataStore, dataSize uint64, stateStore StateStore) (blobstore.BlobAccess, error) {
	readCursor, writeCursor, err := stateStore.Get()
	if err != nil {
		return nil, err
	}

	ba := &circularBlobAccess{
		offsetStore: offsetStore,
		dataStore:   dataStore,
		dataSize:    dataSize,
		stateStore:  stateStore,

		readCursor:  readCursor,
		writeCursor: writeCursor,
	}
	go ba.flushStateStore()
	return ba, nil
}

func (ba *circularBlobAccess) flushStateStore() {
	for {
		time.Sleep(time.Minute)

		ba.lock.Lock()
		readCursor := ba.readCursor
		writeCursor := ba.writeCursor
		ba.lock.Unlock()

		if err := ba.stateStore.Put(readCursor, writeCursor); err != nil {
			log.Print("Failed to write to state store: ", err)
		}
	}
}

func (ba *circularBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	ba.lock.Lock()
	offset, length, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.readCursor, ba.writeCursor)
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
	offset := ba.writeCursor
	ba.writeCursor += uint64(sizeBytes)
	if ba.readCursor > ba.writeCursor {
		ba.readCursor = ba.writeCursor
	} else if ba.readCursor+ba.dataSize < ba.writeCursor {
		ba.readCursor = ba.writeCursor - ba.dataSize
	}
	ba.lock.Unlock()

	// Write the data to storage.
	if err := ba.dataStore.Put(r, offset); err != nil {
		return err
	}

	var err error
	ba.lock.Lock()
	if offset >= ba.readCursor && offset <= ba.writeCursor && offset+uint64(sizeBytes) <= ba.writeCursor {
		err = ba.offsetStore.Put(NewSimpleDigest(digest), offset, sizeBytes, ba.readCursor, ba.writeCursor)
	} else {
		err = errors.New("Data became stale before write completed")
	}
	ba.lock.Unlock()
	return err

}

func (ba *circularBlobAccess) Delete(ctx context.Context, digest *util.Digest) error {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	if offset, _, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.readCursor, ba.writeCursor); err != nil {
		return err
	} else if ok {
		ba.readCursor = offset + 1
		if ba.writeCursor < ba.readCursor {
			ba.writeCursor = ba.readCursor
		}
	}
	return nil
}

func (ba *circularBlobAccess) FindMissing(ctx context.Context, digests []*util.Digest) ([]*util.Digest, error) {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	var missingDigests []*util.Digest
	for _, digest := range digests {
		if _, _, ok, err := ba.offsetStore.Get(NewSimpleDigest(digest), ba.readCursor, ba.writeCursor); err != nil {
			return nil, err
		} else if !ok {
			missingDigests = append(missingDigests, digest)
		}
	}
	return missingDigests, nil
}
