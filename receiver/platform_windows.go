//go:build windows

package main

import (
	"errors"
	"io"
	"os"
	"syscall"
)

// ERROR_NO_DATA (232) is returned when writing to a pipe whose read end has
// been closed. Go's syscall package does not export it as a constant.
const errorNoData = syscall.Errno(232)

// isClosedPipe reports whether a write failed because the consumer closed the
// pipe. socketwrapper/Lyrion closes the socket when playback stops, which is a
// normal way for the receiver to end.
func isClosedPipe(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.ERROR_BROKEN_PIPE) ||
		errors.Is(err, errorNoData)
}
