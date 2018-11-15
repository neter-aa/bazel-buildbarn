package circular

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// maximumIterations denotes the maximum number of changes the
	// Get() and Put() operations may make to the offsets file.
	//
	// Setting this value too high will cause this implementation to
	// become slow under conditions with high hash table collision
	// rates. Conversely, setting this value too low will cause
	// offset entries to be discarded more aggressively, even if the
	// data associated with them is still present in storage.
	//
	// TODO(edsch): Should this option be configurable? If not, is
	// this the right value? Maybe a lower value is sufficient in
	// practice?
	maximumIterations = 8
)

var (
	operationsIterations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore_circular",
			Name:      "file_offset_store_operations_iterations",
			Help:      "Iterations spent per operation on the file offset store.",
			Buckets:   prometheus.LinearBuckets(1.0, 1.0, maximumIterations),
		},
		[]string{"operation", "result"})
)

func init() {
	prometheus.MustRegister(operationsIterations)
}

// offsetRecord contains the hash table entries written to disk. They
// consist of four components:
//
// - A simple digest of the blob (hash and size),
// - The attempt (i.e., how many times this entry got pushed to its next
//   preferential slot in the hash table).
// - The offset of the blob's data within the data file.
// - The length of the blob's data within the data file.
//
// The attempt is part of the record, as it makes it possible to
// distinguish a record from random garbage data. It allows us to
// validate that an entry could have been stored in that location in the
// first place.
type offsetRecord [len(SimpleDigest{}) + 4 + 8 + 8]byte

func newOffsetRecord(digest SimpleDigest, offset uint64, length int64) offsetRecord {
	var offsetRecord offsetRecord
	copy(offsetRecord[:], digest[:])
	binary.LittleEndian.PutUint64(offsetRecord[len(SimpleDigest{})+4:], offset)
	binary.LittleEndian.PutUint64(offsetRecord[len(SimpleDigest{})+4+8:], uint64(length))
	return offsetRecord
}

// getSlot computes the location at which this record should get stored
// within the offset file. It computes an FNV-1a hash from the dige
func (or *offsetRecord) getSlot() uint32 {
	slot := uint32(2166136261)
	for i := len(SimpleDigest{}) + 4; i > 0; i-- {
		slot ^= uint32(or[i-1])
		slot *= 16777619
	}
	return slot
}

func (or *offsetRecord) getAttempt() uint32 {
	return binary.LittleEndian.Uint32(or[len(SimpleDigest{}):])
}

func (or *offsetRecord) getOffset() uint64 {
	return binary.LittleEndian.Uint64(or[len(SimpleDigest{})+4:])
}

func (or *offsetRecord) getLength() int64 {
	return int64(binary.LittleEndian.Uint64(or[len(SimpleDigest{})+4+8:]))
}

func (or *offsetRecord) digestAndAttemptEqual(other offsetRecord) bool {
	return bytes.Equal(or[:len(SimpleDigest{})+4], other[:len(SimpleDigest{})+4])
}

func (or *offsetRecord) offsetAndLengthInBounds(minOffset uint64, maxOffset uint64) bool {
	offset := or.getOffset()
	length := or.getLength()
	return offset >= minOffset && offset <= maxOffset && length >= 0 && offset+uint64(length) <= maxOffset
}

func (or *offsetRecord) withAttempt(attempt uint32) offsetRecord {
	newRecord := *or
	binary.LittleEndian.PutUint32(newRecord[len(SimpleDigest{}):], attempt)
	return newRecord
}

type fileOffsetStore struct {
	file ReadWriterAt
	size uint64
}

// NewFileOffsetStore creates a file-based accessor for the offset
// store. The offset store maps a digest to an offset within the data
// file. This is where the blob's contents may be found.
//
// Under the hood, this implementation uses a hash table with open
// addressing. In order to be self-cleaning, it uses a cuckoo-hash like
// approach, where objects may only be displaced to less preferential
// slots by objects with a higher offset. In other words, more recently
// stored blobs displace older ones.
func NewFileOffsetStore(file ReadWriterAt, size uint64) OffsetStore {
	return &fileOffsetStore{
		file: file,
		size: size,
	}
}

// getPositionOfSlot computes the location at which a hash table slot is
// stored within the offset file.
func (os *fileOffsetStore) getPositionOfSlot(slot uint32) int64 {
	recordLen := uint64(len(offsetRecord{}))
	return int64((uint64(slot) % (os.size / recordLen)) * recordLen)
}

func (os *fileOffsetStore) getRecordAtPosition(position int64) (offsetRecord, error) {
	var record offsetRecord
	if _, err := os.file.ReadAt(record[:], position); err != nil && err != io.EOF {
		return record, err
	}
	return record, nil
}

func (os *fileOffsetStore) putRecordAtPosition(record offsetRecord, position int64) error {
	_, err := os.file.WriteAt(record[:], position)
	return err
}

func (os *fileOffsetStore) Get(digest SimpleDigest, minOffset uint64, maxOffset uint64) (uint64, int64, bool, error) {
	record := newOffsetRecord(digest, 0, 0)
	for iteration := uint32(1); ; iteration++ {
		if iteration >= maximumIterations {
			operationsIterations.WithLabelValues("Get", "TooManyIterations").Observe(float64(iteration))
			return 0, 0, false, nil
		}

		lookupRecord := record.withAttempt(iteration - 1)
		position := os.getPositionOfSlot(lookupRecord.getSlot())
		storedRecord, err := os.getRecordAtPosition(position)
		if err != nil {
			operationsIterations.WithLabelValues("Get", "Error").Observe(float64(iteration))
			return 0, 0, false, err
		}
		if !storedRecord.offsetAndLengthInBounds(minOffset, maxOffset) {
			operationsIterations.WithLabelValues("Get", "NotFound").Observe(float64(iteration))
			return 0, 0, false, nil
		}
		if storedRecord.digestAndAttemptEqual(lookupRecord) {
			operationsIterations.WithLabelValues("Get", "Success").Observe(float64(iteration))
			return storedRecord.getOffset(), storedRecord.getLength(), true, nil
		}
		if os.getPositionOfSlot(storedRecord.getSlot()) != position {
			operationsIterations.WithLabelValues("Get", "NotFound").Observe(float64(iteration))
			return 0, 0, false, nil
		}
	}
}

func (os *fileOffsetStore) putRecord(record offsetRecord, minOffset uint64, maxOffset uint64) (offsetRecord, bool, error) {
	position := os.getPositionOfSlot(record.getSlot())

	// Fetch the old record. If it is invalid, or already at a spot
	// where it can't be moved to another place, simply overwrite it.
	oldRecord, err := os.getRecordAtPosition(position)
	if err != nil {
		return offsetRecord{}, false, err
	}
	oldAttempt := oldRecord.getAttempt()
	if !oldRecord.offsetAndLengthInBounds(minOffset, maxOffset) ||
		oldAttempt >= maximumIterations-1 ||
		os.getPositionOfSlot(oldRecord.getSlot()) != position {
		// Record at this position is invalid/outdated.
		// Overwrite it.
		return offsetRecord{}, false, os.putRecordAtPosition(record, position)
	}

	if oldRecord.getOffset() <= record.getOffset() {
		// Record is valid, but older than the one we're
		// inserting. Displace the old record to its next slot.
		return oldRecord.withAttempt(oldAttempt + 1), true, os.putRecordAtPosition(record, position)
	}

	// Record is newer than the one we're inserting. See if we still
	// have another place to put it.
	attempt := record.getAttempt()
	if attempt >= maximumIterations-1 {
		return offsetRecord{}, false, nil
	}
	return record.withAttempt(attempt + 1), true, nil
}

func (os *fileOffsetStore) Put(digest SimpleDigest, offset uint64, length int64, minOffset uint64, maxOffset uint64) error {
	// Insert the new record. Doing this may yield another that got
	// displaced. Iteratively try to re-insert those.
	record := newOffsetRecord(digest, offset, length)
	for iteration := 1; ; iteration++ {
		if iteration > maximumIterations {
			operationsIterations.WithLabelValues("Put", "TooManyIterations").Observe(float64(iteration))
			return nil
		}

		if nextRecord, more, err := os.putRecord(record, minOffset, maxOffset); err != nil {
			operationsIterations.WithLabelValues("Put", "Error").Observe(float64(iteration))
			return err
		} else if more {
			record = nextRecord
		} else {
			operationsIterations.WithLabelValues("Put", "Success").Observe(float64(iteration))
			return nil
		}
	}
}
