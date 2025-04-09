package filterliststore

import (
	"io"
)

// notifyReadCloser is an io.ReadCloser that notifies a channel when it is closed.
type notifyReadCloser struct {
	// reader is the underlying io.ReadCloser being wrapped.
	reader io.ReadCloser
	// notifyCh is the channel to which a notification is sent when the reader is closed.
	notifyCh chan<- struct{}
	// closed indicates whether the reader has been closed.
	closed bool
}

func newNotifyReadCloser(reader io.ReadCloser) (*notifyReadCloser, <-chan struct{}) {
	notifyCh := make(chan struct{}, 1) // Buffered channel to avoid blocking.
	return &notifyReadCloser{
		reader:   reader,
		notifyCh: notifyCh,
	}, notifyCh
}

func (n *notifyReadCloser) Read(p []byte) (int, error) {
	return n.reader.Read(p)
}

func (n *notifyReadCloser) Close() error {
	if n.closed {
		return nil
	}
	n.closed = true
	err := n.reader.Close()
	n.notifyCh <- struct{}{}
	if err != nil {
		return err
	}
	return nil
}
