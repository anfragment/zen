package htmlrewrite

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/html/charset"
)

// readRawBody extracts an uncompressed, UTF-8 decoded body from a potentially compressed and non-UTF-8 encoded HTTP response.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func readRawBody(res *http.Response) ([]byte, error) {
	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	res.Body.Close()

	decompressedReader, err := decompressReader(bytes.NewReader(resBytes), res.Header.Get("Content-Encoding"))
	if err != nil {
		res.Body = io.NopCloser(bytes.NewReader(resBytes))
		return nil, fmt.Errorf("create decompressed reader: %w", err)
	}

	contentType := res.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		res.Body = io.NopCloser(bytes.NewReader(resBytes))
		return nil, fmt.Errorf("parse media type: %w", err)
	}
	decodedReader, err := decodeReader(decompressedReader, params["charset"])
	if err != nil {
		decompressedReader.Close()
		res.Body = io.NopCloser(bytes.NewReader(resBytes))
		return nil, fmt.Errorf("create decoded reader: %w", err)
	}

	originalBody, err := io.ReadAll(decodedReader)
	decompressedReader.Close()
	if err != nil {
		res.Body = io.NopCloser(bytes.NewReader(resBytes))
		return nil, fmt.Errorf("read decompressed+decoded stream: %w", err)
	}
	return originalBody, nil
}

// decompressReader decompresses a reader using the specified compression algorithm.
// It does not decompress data encoded with multiple algorithms.
func decompressReader(reader io.Reader, compressionAlg string) (io.ReadCloser, error) {
	// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Encoding
	switch strings.ToLower(compressionAlg) {
	case "gzip":
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
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
			return nil, fmt.Errorf("create zstd reader: %w", err)
		}
		return io.NopCloser(zstdReader), nil
	case "":
		return io.NopCloser(reader), nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", compressionAlg)
	}
}

// decodeReader decodes the reader based on the provided character encoding.
func decodeReader(reader io.Reader, encoding string) (io.Reader, error) {
	// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Type#media-type
	switch strings.ToLower(encoding) {
	case "utf-8", "us-ascii", "":
		return reader, nil
	default:
		encoding, _ := charset.Lookup(encoding)
		if encoding == nil {
			return nil, fmt.Errorf("unsupported charset %q", encoding)
		}
		return encoding.NewDecoder().Reader(reader), nil
	}
}
