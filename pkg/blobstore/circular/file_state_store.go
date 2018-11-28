package circular

import (
	"encoding/binary"
	"io"
	"log"
)

type fileStateStore struct {
	file ReadWriterAt
}

// NewFileStateStore creates a new storage for global metadata of a
// circular storage backend. Right now only a set of read/write cursors
// are stored.
func NewFileStateStore(file ReadWriterAt) StateStore {
	return &fileStateStore{
		file: file,
	}
}

func (ss *fileStateStore) Get() (Cursors, error) {
	var data [16]byte
	if _, err := ss.file.ReadAt(data[:], 0); err == io.EOF {
		return Cursors{}, nil
	} else if err != nil {
		return Cursors{}, err
	}
	readCursor := binary.LittleEndian.Uint64(data[:])
	writeCursor := binary.LittleEndian.Uint64(data[8:])
	if readCursor > writeCursor {
		return Cursors{}, nil
	}
	return Cursors{
		Read:  readCursor,
		Write: writeCursor,
	}, nil
}

func (ss *fileStateStore) Put(cursors Cursors) error {
	if cursors.Read > cursors.Write {
		log.Fatalf("Attempted to write cursors %d > %d", cursors.Read, cursors.Write)
	}
	var data [16]byte
	binary.LittleEndian.PutUint64(data[:], cursors.Read)
	binary.LittleEndian.PutUint64(data[8:], cursors.Write)
	_, err := ss.file.WriteAt(data[:], 0)
	return err
}
