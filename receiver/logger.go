// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
package main

import (
	"fmt"
	"os"
)

// Log levels follow quiniouben/vban: 0 fatal, 1 error, 2 info, 3 debug.
const (
	levelFatal = 0
	levelError = 1
	levelInfo  = 2
	levelDebug = 3
)

type logger struct {
	level int
}

func newLogger(level int) *logger {
	return &logger{level: level}
}

func (l *logger) logf(level int, prefix, format string, args ...any) {
	if l.level < level {
		return
	}
	fmt.Fprintf(os.Stderr, "vban_url_receptor %s: %s\n", prefix, fmt.Sprintf(format, args...))
}

func (l *logger) fatal(format string, args ...any) { l.logf(levelFatal, "fatal", format, args...) }
func (l *logger) error(format string, args ...any) { l.logf(levelError, "error", format, args...) }
func (l *logger) info(format string, args ...any)  { l.logf(levelInfo, "info", format, args...) }
func (l *logger) debug(format string, args ...any) { l.logf(levelDebug, "debug", format, args...) }
