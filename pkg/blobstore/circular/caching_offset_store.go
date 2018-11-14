package circular

import (
	"encoding/binary"
)

type cachedRecord struct {
	digest SimpleDigest
	offset uint64
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

func (os *cachingOffsetStore) Get(digest SimpleDigest, minOffset uint64, maxOffset uint64) (uint64, bool, error) {
	slot := binary.LittleEndian.Uint32(digest[:]) % uint32(len(os.table))
	foundRecord := os.table[slot]
	if foundRecord.digest == digest && foundRecord.offset >= minOffset && foundRecord.offset <= maxOffset {
		return foundRecord.offset, true, nil
	}

	offset, found, err := os.backend.Get(digest, minOffset, maxOffset)
	if err == nil && found {
		os.table[slot] = cachedRecord{
			digest: digest,
			offset: offset,
		}
	}
	return offset, found, err
}

func (os *cachingOffsetStore) Put(digest SimpleDigest, offset uint64, minOffset uint64, maxOffset uint64) error {
	if err := os.backend.Put(digest, offset, minOffset, maxOffset); err != nil {
		return err
	}

	slot := binary.LittleEndian.Uint32(digest[:]) % uint32(len(os.table))
	os.table[slot] = cachedRecord{
		digest: digest,
		offset: offset,
	}
	return nil
}
