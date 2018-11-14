package circular

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type cachedRecord struct {
	hash      [sha256.Size]byte
	sizeBytes int64
	offset    uint64
}

func newCachedRecord(digest *util.Digest) cachedRecord {
	cr := cachedRecord{
		sizeBytes: digest.GetSizeBytes(),
	}
	copy(cr.hash[:sha256.Size], digest.GetHash())
	return cr
}

func (cr *cachedRecord) getSlot() uint32 {
	return binary.LittleEndian.Uint32(cr.hash[:])
}

func (cr *cachedRecord) digestEqual(other cachedRecord) bool {
	return cr.hash == other.hash && cr.sizeBytes == other.sizeBytes
}

func (cr *cachedRecord) offsetInBounds(minOffset uint64, maxOffset uint64) bool {
	return cr.offset >= minOffset && cr.offset <= maxOffset
}

func (cr *cachedRecord) withOffset(offset uint64) cachedRecord {
	newRecord := *cr
	newRecord.offset = offset
	return newRecord
}

type cachingOffsetStore struct {
	backend OffsetStore
	table   []cachedRecord
}

func NewCachingOffsetStore(backend OffsetStore, size uint) OffsetStore {
	return &cachingOffsetStore{
		backend: backend,
		table:   make([]cachedRecord, size),
	}
}

func (os *cachingOffsetStore) Get(digest *util.Digest, minOffset uint64, maxOffset uint64) (uint64, bool, error) {
	record := newCachedRecord(digest)
	slot := record.getSlot() % uint32(len(os.table))
	foundRecord := os.table[slot]
	if foundRecord.digestEqual(record) && foundRecord.offsetInBounds(minOffset, maxOffset) {
		return foundRecord.offset, true, nil
	}

	offset, found, err := os.backend.Get(digest, minOffset, maxOffset)
	if err == nil && found {
		os.table[slot] = record.withOffset(offset)
	}
	return offset, found, err
}

func (os *cachingOffsetStore) Put(digest *util.Digest, offset uint64, minOffset uint64, maxOffset uint64) error {
	if err := os.backend.Put(digest, offset, minOffset, maxOffset); err != nil {
		return err
	}

	record := newCachedRecord(digest)
	slot := record.getSlot() % uint32(len(os.table))
	os.table[slot] = record.withOffset(offset)
	return nil
}
