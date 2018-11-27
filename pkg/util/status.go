package util

import (
	"fmt"

	"google.golang.org/grpc/status"
)

// StatusWrap prepends a string to the message of an existing error.
func StatusWrap(err error, msg string) error {
	p := status.Convert(err).Proto()
	p.Message = fmt.Sprintf("%s: %s", msg, p.Message)
	return status.ErrorProto(p)
}

// StatusWrapf prepends a formatted string to the message of an existing error.
func StatusWrapf(err error, format string, args ...interface{}) error {
	return StatusWrap(err, fmt.Sprintf(format, args...))
}
