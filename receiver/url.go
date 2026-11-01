// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
//
// This file is part of vban-squeeze-win, a Windows port of
// https://github.com/markusbuschhoff/vban-squeeze (GPL-3.0).
package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	defaultHost   = "127.0.0.1"
	defaultPort   = 6980
	defaultStream = "Stream1"

	// vbanStreamNameSize is the size of the stream name field in a VBAN header.
	vbanStreamNameSize = 16
)

// streamURL is the parsed form of vban://[host[:port]][/streamname].
type streamURL struct {
	Host   string
	Port   int
	Stream string
}

func (u streamURL) String() string {
	return fmt.Sprintf("vban://%s:%d/%s", u.Host, u.Port, u.Stream)
}

// parseURL parses the URL scheme used by the plugin. Host and port are only
// informational: the receiver binds 0.0.0.0 and filters on the stream name,
// exactly like vban_receptor from quiniouben/vban does.
func parseURL(raw string) (streamURL, error) {
	u := streamURL{Host: defaultHost, Port: defaultPort, Stream: defaultStream}

	s := strings.TrimSpace(raw)
	if s == "" {
		return u, errors.New("empty URL")
	}

	if !strings.Contains(s, "://") {
		s = "vban://" + strings.TrimLeft(s, "/")
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return u, fmt.Errorf("invalid URL %q: %w", raw, err)
	}

	if !strings.EqualFold(parsed.Scheme, "vban") {
		return u, fmt.Errorf("unsupported scheme %q, expected vban://", parsed.Scheme)
	}

	if host := parsed.Hostname(); host != "" {
		u.Host = host
	}

	if p := parsed.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return u, fmt.Errorf("invalid port %q", p)
		}
		u.Port = n
	}

	if stream := strings.TrimPrefix(parsed.EscapedPath(), "/"); stream != "" {
		if decoded, err := url.PathUnescape(stream); err == nil {
			stream = decoded
		}
		u.Stream = stream
	}

	if u.Port < 1 || u.Port > 65535 {
		return u, fmt.Errorf("port %d out of range", u.Port)
	}

	if len(u.Stream) > vbanStreamNameSize {
		return u, fmt.Errorf("stream name %q is longer than %d bytes", u.Stream, vbanStreamNameSize)
	}

	return u, nil
}
