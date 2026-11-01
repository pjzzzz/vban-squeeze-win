//go:build !windows

package main

import (
	"errors"
	"io"
	"net"
	"os"
	"syscall"
)

func isClosedPipe(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.EPIPE)
}

func reuseAddrControl(_, _ string, c syscall.RawConn) error {
	var sockErr error
	if err := c.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	}); err != nil {
		return err
	}
	return sockErr
}

func joinMulticastGroup(conn *net.UDPConn, group, iface net.IP) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		mreq := &syscall.IPMreq{}
		copy(mreq.Multiaddr[:], group.To4())
		if iface != nil {
			copy(mreq.Interface[:], iface.To4())
		}
		sockErr = syscall.SetsockoptIPMreq(int(fd), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq)
	}); err != nil {
		return err
	}
	return sockErr
}
