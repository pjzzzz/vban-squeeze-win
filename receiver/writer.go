// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
package main

import "io"

// asyncWriter decouples reading/measuring from the (blocking) PCM output.
// Without it, Lyrion's backpressure would throttle the UDP read loop, making a
// 48000 Hz stream look like 44100 Hz and breaking the rate measurement.
type asyncWriter struct {
	ch     chan []byte
	errCh  chan error
	closed chan struct{}
}

func newAsyncWriter(w io.Writer) *asyncWriter {
	a := &asyncWriter{
		ch:     make(chan []byte, 512),
		errCh:  make(chan error, 1),
		closed: make(chan struct{}),
	}

	go func() {
		defer close(a.closed)
		for b := range a.ch {
			if _, err := w.Write(b); err != nil {
				a.errCh <- err
				return
			}
		}
	}()

	return a
}

func (a *asyncWriter) write(b []byte) error {
	select {
	case a.ch <- b:
		return nil
	case err := <-a.errCh:
		return err
	case <-a.closed:
		return io.ErrClosedPipe
	}
}

func (a *asyncWriter) close() {
	close(a.ch)
	<-a.closed
}
