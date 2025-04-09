package filterliststore

import (
	"errors"
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

func newEavesdropReadCloser(reader io.ReadCloser) (*eavesdropReadCloser, <-chan []byte, error) {
	if reader == nil {
		return nil, nil, errors.New("reader cannot be nil")
	}

	resCh := make(chan []byte, 1) // Buffered channel to avoid blocking
	return &eavesdropReadCloser{
		reader: reader,
		resCh:  resCh,
		buf:    make([]byte, 0),
	}, resCh, nil
}

var _ io.ReadCloser = (*eavesdropReadCloser)(nil)

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
