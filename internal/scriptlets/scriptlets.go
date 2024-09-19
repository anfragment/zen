package scriptlets

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/html/charset"
)

//go:embed bundle.js
var scriptletsBundleFS embed.FS

// reBody captures contents of the body tag in an HTML document.
var reBody = regexp.MustCompile(`(?i)<body[\s\S]*?>([\s\S]*)</body>`)

// Injector injects scriptlets into HTML HTTP responses.
type Injector struct {
	// scriptletsElement contains the <script> element for the scriptlets bundle, which will be inserted into HTML documents.
	scriptletsElement []byte
}

// NewInjector creates a new Injector with the embedded scriptlets.
func NewInjector() (*Injector, error) {
	bundleData, err := scriptletsBundleFS.ReadFile("bundle.js")
	if err != nil {
		return nil, fmt.Errorf("read bundle from embed: %w", err)
	}

	openingTag := []byte("\n<script>")
	closingTag := []byte("</script>")

	scriptletsElement := make([]byte, len(openingTag)+len(bundleData)+len(closingTag))
	copy(scriptletsElement, openingTag)
	copy(scriptletsElement[len(openingTag):], bundleData)
	copy(scriptletsElement[len(openingTag)+len(bundleData):], closingTag)

	return &Injector{
		scriptletsElement: scriptletsElement,
	}, nil
}

// Inject injects scriptlets into given HTTP HTML response.
//
// In case of error, the response body is unchanged and the caller may proceed as if the function had not been called.
func (m *Injector) Inject(req *http.Request, res *http.Response) error {
	rawBodyBytes, err := readRawBody(res)
	if err != nil {
		return fmt.Errorf("read raw body: %w", err)
	}

	var modified bool
	modifiedBody := reBody.ReplaceAllFunc(rawBodyBytes, func(match []byte) []byte {
		modified = true
		return append(match, m.scriptletsElement...)
	})

	if !modified {
		log.Printf("no body tag found in response from %q", req.URL)
	}

	res.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	res.ContentLength = int64(len(modifiedBody))
	res.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))
	res.Header.Del("Content-Encoding")
	res.Header.Set("Content-Type", "text/html; charset=utf-8")

	return nil
}

// readRawBody extracts a raw body from a potentially compressed and non-UTF8 encoded HTTP response.
//
// In case of an error, the response body is unchanged and the caller may proceed as if the function had not been called.
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

// decompressReader decompresses the reader based on the provided compression algorithm.
// It does not decompress data encoded with multiple algorithms.
//
// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Encoding
func decompressReader(reader io.Reader, compressionAlg string) (io.ReadCloser, error) {
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
//
// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Type#media-type
func decodeReader(reader io.Reader, encoding string) (io.Reader, error) {
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
