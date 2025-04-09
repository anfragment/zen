package filterliststore

import (
	"io"
)

// eavesdropReadCloser is a wrapper around io.ReadCloser that captures
// the data read from the underlying reader and sends it to a channel
// after the reader is closed.
type eavesdropReadCloser struct {
	// reader is the underlying io.ReadCloser being wrapped.
	reader io.ReadCloser
	// resCh is the channel to which the captured data is sent.
	resCh chan<- []byte
	// buf accumulates the data read from reader.
	buf []byte
}

func newEavesdropReadCloser(reader io.ReadCloser) (*eavesdropReadCloser, <-chan []byte) {
	resCh := make(chan []byte, 1) // Buffered channel to avoid blocking
	return &eavesdropReadCloser{
		reader: reader,
		resCh:  resCh,
		buf:    make([]byte, 0),
	}, resCh
}

func (e *eavesdropReadCloser) Read(p []byte) (n int, err error) {
	n, err = e.reader.Read(p)
	e.buf = append(e.buf, p[:n]...)
	return n, err
}

func (e *eavesdropReadCloser) Close() error {
	err := e.reader.Close()
	e.resCh <- e.buf
	return err
}

// readThenCloseReadCloser is an io.ReadCloser that reads from r1 until EOF,
// then continues reading from r2. Close only calls r2.Close.
type readThenCloseReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func newReadThenCloseReadCloser(r1 io.Reader, r2 io.ReadCloser) io.ReadCloser {
	return &readThenCloseReadCloser{
		reader: io.MultiReader(r1, r2),
		closer: r2,
	}
}

func (rc *readThenCloseReadCloser) Read(p []byte) (int, error) {
	return rc.reader.Read(p)
}

func (rc *readThenCloseReadCloser) Close() error {
	return rc.closer.Close()
}
