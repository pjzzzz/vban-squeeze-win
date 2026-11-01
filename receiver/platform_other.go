//go:build !windows

package main

import (
	"errors"
	"io"
	"os"
	"syscall"
)

func isClosedPipe(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.EPIPE)
}
