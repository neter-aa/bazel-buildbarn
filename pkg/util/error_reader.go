package util

import (
	"io"
)

// NewErrorReader creates a ReadCloser that returns a fixed error upon
// reads. This is primarily used by implementations of BlobAccess to
// return errors for Get().
func NewErrorReader(err error) io.ReadCloser {
	return &errorReader{err: err}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func (r *errorReader) Close() error {
	return nil
}
