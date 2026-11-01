//go:build windows

package main

import (
	"errors"
	"io"
	"net"
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

// reuseAddrControl sets SO_REUSEADDR. Only used for multicast listeners: it
// lets several receivers on the same host bind the same UDP port, and every
// socket that joined the group receives its own copy of each datagram.
func reuseAddrControl(_, _ string, c syscall.RawConn) error {
	var sockErr error
	if err := c.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	}); err != nil {
		return err
	}
	return sockErr
}

// joinMulticastGroup adds an IP_ADD_MEMBERSHIP for group on the socket. A nil
// iface lets Windows pick the default multicast interface.
func joinMulticastGroup(conn *net.UDPConn, group, iface net.IP) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		mreq := make([]byte, 8) // struct ip_mreq: multiaddr + interface
		copy(mreq[0:4], group.To4())
		if iface != nil {
			copy(mreq[4:8], iface.To4())
		}
		sockErr = syscall.Setsockopt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, &mreq[0], int32(len(mreq)))
	}); err != nil {
		return err
	}
	return sockErr
}
