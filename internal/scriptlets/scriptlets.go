package scriptlets

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html/charset"
)

//go:embed bundle.js
var scriptletsBundleFS embed.FS

// reBody matches contents of the body tag in an HTML document.
var reBody = regexp.MustCompile(`(?i)<body>([\s\S]*)</body>`)

type Injector struct {
	// scriptletsElement contains the <script> element for the scriptlets bundle, which will be inserted into HTML documents.
	scriptletsElement []byte
}

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
		scriptletsElement: bundleData,
	}, nil
}

func (m *Injector) Inject(req *http.Request, res *http.Response) error {
	originalBody, err := readResponseBody(res)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	res.Body.Close()

	var modified bool
	modifiedBody := reBody.ReplaceAllFunc(originalBody, func(match []byte) []byte {
		modified = true
		return append(match, m.scriptletsElement...)
	})

	if !modified {
		log.Printf("no body tag found in response from %q\n%s", req.URL, originalBody)
	}

	res.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	res.ContentLength = int64(len(modifiedBody))
	res.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))

	return nil
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %v", err)
	}
	resp.Body.Close()

	// Handle decompression if needed.
	var decompressedBytes []byte
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch strings.ToLower(contentEncoding) {
	case "gzip":
		gzipReader, err := gzip.NewReader(bytes.NewReader(respBytes))
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		decompressedBytes, err = ioutil.ReadAll(gzipReader)
		if err != nil {
			return nil, fmt.Errorf("read gzip content: %w", err)
		}
	case "deflate":
		flateReader := flate.NewReader(bytes.NewReader(respBytes))
		defer flateReader.Close()
		decompressedBytes, err = ioutil.ReadAll(flateReader)
		if err != nil {
			return nil, fmt.Errorf("read deflate content: %w", err)
		}
	case "br":
		brotliReader := brotli.NewReader(bytes.NewReader(respBytes))
		decompressedBytes, err = ioutil.ReadAll(brotliReader)
		if err != nil {
			return nil, fmt.Errorf("read brotli content: %w", err)
		}
	case "", "identity":
		// No compression.
		decompressedBytes = respBytes
	default:
		return nil, fmt.Errorf("unsupported content encoding: %s", contentEncoding)
	}

	// Parse the Content-Type Header to get the charset
	contentType := resp.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("parse media type: %w", err)
	}

	// Default to UTF-8 if no charset is specified.
	charsetStr := strings.ToLower(params["charset"])
	if charsetStr == "" {
		charsetStr = "utf-8"
	}

	// Create a reader that decodes the response body according to the charset.
	var finalReader io.Reader = bytes.NewReader(decompressedBytes)
	if charsetStr != "utf-8" && charsetStr != "us-ascii" {
		// Use the charset to create a decoder
		encoding, _ := charset.Lookup(charsetStr)
		if encoding == nil {
			return nil, fmt.Errorf("unsupported charset: %s", charsetStr)
		}
		finalReader = encoding.NewDecoder().Reader(finalReader)
	}

	bodyBytes, err := ioutil.ReadAll(finalReader)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return bodyBytes, nil
}
