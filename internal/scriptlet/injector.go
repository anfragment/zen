package scriptlet

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"embed"
	"errors"
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

var (
	//go:embed bundle.js
	scriptletsBundleFS embed.FS
	// reBody captures contents of the body tag in an HTML document.
	reBody           = regexp.MustCompile(`(?i)<body[\s\S]*?>([\s\S]*)</body>`)
	scriptOpeningTag = []byte("<script>")
	scriptClosingTag = []byte("</script>")
)

type Store interface {
	Add(hostnames []string, scriptlet *Scriptlet)
	Get(hostname string) []*Scriptlet
}

// Injector injects scriptlets into HTML HTTP responses.
type Injector struct {
	// bundle contains the <script> element for the scriptlets bundle, which is to be inserted into HTML documents.
	bundle []byte
	// store stores and retrieves scriptlets by hostname.
	store Store
}

// NewInjector creates a new Injector with the embedded scriptlets.
func NewInjector(store Store) (*Injector, error) {
	if store == nil {
		return nil, errors.New("store is nil")
	}

	bundleData, err := scriptletsBundleFS.ReadFile("bundle.js")
	if err != nil {
		return nil, fmt.Errorf("read bundle from embed: %w", err)
	}

	scriptletsElement := make([]byte, len(scriptOpeningTag)+len(bundleData)+len(scriptClosingTag))
	copy(scriptletsElement, scriptOpeningTag)
	copy(scriptletsElement[len(scriptOpeningTag):], bundleData)
	copy(scriptletsElement[len(scriptOpeningTag)+len(bundleData):], scriptClosingTag)

	return &Injector{
		bundle: scriptletsElement,
		store:  store,
	}, nil
}

// Inject injects scriptlets into a given HTTP HTML response.
//
// In case of an error, the response body is unchanged and the caller may proceed as if the function had not been called.
func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	scriptlets := inj.store.Get(req.URL.Hostname())
	log.Printf("got %d scriptlets for %q", len(scriptlets), req.URL.Hostname())
	if len(scriptlets) == 0 {
		return nil
	}
	var ruleInjection bytes.Buffer
	ruleInjection.Write(scriptOpeningTag)
	ruleInjection.WriteString("\n(function() {\n")
	var err error
	for _, scriptlet := range scriptlets {
		if err = scriptlet.GenerateInjection(&ruleInjection); err != nil {
			return fmt.Errorf("generate injection for scriptlet %q: %w", scriptlet.Name, err)
		}
		ruleInjection.WriteByte('\n')
	}
	ruleInjection.WriteString("})();\n")
	ruleInjection.Write(scriptClosingTag)

	rawBodyBytes, err := readRawBody(res)
	if err != nil {
		return fmt.Errorf("read raw body: %w", err)
	}

	var modified bool
	modifiedBody := reBody.ReplaceAllFunc(rawBodyBytes, func(match []byte) []byte {
		modified = true
		match = append(match, inj.bundle...)
		match = append(match, '\n')
		match = append(match, ruleInjection.Bytes()...)
		return match
	})

	if !modified {
		return nil
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
