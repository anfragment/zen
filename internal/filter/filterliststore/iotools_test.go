package filterliststore

import (
	"bytes"
	"io"
	"testing"
)

func TestEavesdropReader(t *testing.T) {
	t.Parallel()

	original := []byte("making rocks think was a mistake")
	reader := bytes.NewReader(original)

	wrappedReader, resCh := newEavesdropReadCloser(io.NopCloser(reader))

	go func() {
		readBytes, err := io.ReadAll(wrappedReader)
		if err != nil {
			t.Errorf("failed to read from eavesdropped reader: %v", err)
		}
		if err := wrappedReader.Close(); err != nil {
			t.Errorf("failed to close wrapped reader: %v", err)
		}
		if !bytes.Equal(readBytes, original) {
			t.Errorf("expected wrapped reader to yield %q, got %q", original, readBytes)
		}
	}()

	eavesdropped := <-resCh
	if !bytes.Equal(eavesdropped, original) {
		t.Errorf("expected eavesdropped bytes to be %q, got %q", original, eavesdropped)
	}
}
