package htmlrewrite

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/hashicorp/go-multierror"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/html/charset"
)

// getRawBodyReader extracts an uncompressed, UTF-8 decoded body from a potentially compressed and non-UTF-8 encoded HTTP response.
func getRawBodyReader(res *http.Response) (io.ReadCloser, error) {
	encoding := res.Header.Get("Content-Encoding")
	contentType := res.Header.Get("Content-Type")
	if encoding == "" && strings.Contains(contentType, "charset=utf-8") {
		// We've been here before, skip costly operations.
		return res.Body, nil
	}

	decompressedReader, err := decompressReader(res.Body, encoding)
	if err != nil {
		return nil, fmt.Errorf("create decompressed reader for encoding %q: %w", encoding, err)
	}

	decodedReader, err := charset.NewReader(decompressedReader, contentType)
	if err != nil {
		decompressedReader.Close()
		return nil, fmt.Errorf("create decoded reader for content type %q: %w", contentType, err)
	}

	return struct {
		io.Reader
		io.Closer
	}{
		decodedReader,
		&multiCloser{[]io.Closer{decompressedReader, res.Body}},
	}, nil
}

// decompressReader decompresses a reader using the specified compression algorithm.
// It does not decompress data encoded with multiple algorithms.
func decompressReader(reader io.ReadCloser, compressionAlg string) (io.ReadCloser, error) {
	// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Encoding
	switch strings.ToLower(compressionAlg) {
	case "gzip":
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		return gzipReader, nil
	case "deflate":
		return flate.NewReader(reader), nil
	case "br":
		return io.NopCloser(brotli.NewReader(reader)), nil
	case "zstd":
		zstdReader, err := zstd.NewReader(reader)
		if err != nil {
			zstdReader.Close()
			return nil, fmt.Errorf("create zstd reader: %w", err)
		}
		return io.NopCloser(zstdReader), nil
	case "":
		return reader, nil
	default:
		return nil, errors.New("unsupported encoding")
	}
}

// multiCloser wraps multiple io.Closers and ensures they are closed sequentially.
type multiCloser struct {
	closers []io.Closer
}

// Close iterates over each io.Closer and closes it, capturing any errors.
func (m *multiCloser) Close() error {
	var finalErr *multierror.Error
	for _, closer := range m.closers {
		if err := closer.Close(); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}
	return finalErr.ErrorOrNil()
}
