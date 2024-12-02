package cosmetic

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/anfragment/zen/internal/logger"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/html/charset"
)

var (
	// reBody captures contents of the body tag in an HTML document.
	reBody          = regexp.MustCompile(`(?i)<body[\s\S]*?>([\s\S]*)</body>`)
	styleOpeningTag = []byte("<style>")
	styleClosingTag = []byte("</style>")
)

type Injector struct {
	// store stores and retrieves css by hostname.
	store Store
}

type Store interface {
	Add(hostnames []string, selector string)
	Get(hostname string) []string
}

func NewInjector(store Store) (*Injector, error) {
	if store == nil {
		return nil, errors.New("store is nil")
	}

	return &Injector{
		store: store,
	}, nil
}

func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	hostname := req.URL.Hostname()
	log.Printf("injecting css for %q\n", logger.Redacted(hostname))

	selectors := inj.store.Get(hostname)
	log.Printf("got %d selectors for %q", len(selectors), logger.Redacted(hostname))
	if len(selectors) == 0 {
		return nil
	}

	var ruleInjection bytes.Buffer
	ruleInjection.Write(styleOpeningTag)
	for _, selector := range selectors {
		ruleInjection.WriteString(fmt.Sprintf("%s { display: none !important; }\n", selector))
	}
	ruleInjection.Write(styleClosingTag)

	rawBodyBytes, err := readRawBody(res)
	if err != nil {
		return fmt.Errorf("read raw body: %w", err)
	}

	modifiedBody := reBody.ReplaceAllFunc(rawBodyBytes, func(body []byte) []byte {
		// Inject ruleInjection right after the <body> tag.
		return bytes.Join([][]byte{body, ruleInjection.Bytes()}, nil)
	})

	res.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	res.ContentLength = int64(len(modifiedBody))
	res.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))
	res.Header.Del("Content-Encoding")
	res.Header.Set("Content-Type", "text/html; charset=utf-8")

	return nil
}

// readRawBody is replicated from internal/filter/scriptlet/injector.go for now.
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
