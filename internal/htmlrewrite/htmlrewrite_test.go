package htmlrewrite_test

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"testing"

	"github.com/anfragment/zen/internal/htmlrewrite"
)

func TestAppendHeadContentsPublic(t *testing.T) {
	t.Parallel()

	type tc struct {
		name       string
		original   []byte
		appendWith []byte
		expected   []byte
	}

	tests := []tc{
		{
			"appends <head> contents",
			[]byte(`<html><head>Original Head Content</head><body>Test</body></html>`),
			[]byte("-Appended"),
			[]byte(`<html><head>Original Head Content-Appended</head><body>Test</body></html>`),
		},
		{
			"doesn't modify body on empty appendWith",
			[]byte(`<html><head>Original Head Content</head></html>`),
			[]byte(""),
			[]byte(`<html><head>Original Head Content</head></html>`),
		},
		{
			"doesn't modify response if no <head> is present",
			[]byte(`<html><body>Test</body></html>`),
			[]byte("test"),
			[]byte(`<html><body>Test</body></html>`),
		},
	}

	generatedBytes := genAlphanumByteArray(10 * 1024 * 1024) // 10MB
	tests = append(tests, tc{
		name:       "appends to large <head> contents",
		original:   bytes.Join([][]byte{[]byte(`<html><head>`), generatedBytes, []byte(`</head></html>`)}, nil),
		appendWith: []byte("-Appended"),
		expected:   bytes.Join([][]byte{[]byte(`<html><head>`), generatedBytes, []byte("-Appended"), []byte(`</head></html>`)}, nil),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := newHTTPResponse(tt.original)
			if err := htmlrewrite.AppendHeadContents(res, tt.appendWith); err != nil {
				t.Fatalf("AppendHeadContents error: %v", err)
			}
			modifiedBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read modified body: %v", err)
			}
			if !bytes.Equal(modifiedBody, tt.expected) {
				if len(modifiedBody) < 1024 && len(tt.expected) < 1024 {
					t.Errorf("expected response body %q, got %q", tt.expected, modifiedBody)
				} else {
					t.Error("expected body != modifiedBody")
				}
			}
		})
	}
}

func TestPrependBodyContentsPublic(t *testing.T) {
	t.Parallel()

	type tc struct {
		name        string
		original    []byte
		prependWith []byte
		expected    []byte
	}

	tests := []tc{
		{
			"prepends <body> contents",
			[]byte(`<html><head>Test</head><body>Original Body Content</body></html>`),
			[]byte("Non-"),
			[]byte(`<html><head>Test</head><body>Non-Original Body Content</body></html>`),
		},
		{
			"prepends <body> contents with empty byte array",
			[]byte(`<html><body>Original Body Content</body></html>`),
			[]byte(""),
			[]byte(`<html><body>Original Body Content</body></html>`),
		},
		{
			"doesn't modify response if no <body> is present",
			[]byte(`<html></html>`),
			[]byte("test"),
			[]byte(`<html></html>`),
		},
	}

	generatedBytes := genAlphanumByteArray(10 * 1024 * 1024) // 10MB
	tests = append(tests, tc{
		name:        "prepends to large <body> contents",
		original:    bytes.Join([][]byte{[]byte(`<html><body>`), generatedBytes, []byte(`</body></html>`)}, nil),
		prependWith: []byte("Prepended"),
		expected:    bytes.Join([][]byte{[]byte(`<html><body>`), []byte("Prepended"), generatedBytes, []byte(`</body></html>`)}, nil),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := newHTTPResponse(tt.original)
			if err := htmlrewrite.PrependBodyContents(res, tt.prependWith); err != nil {
				t.Fatalf("PrependBodyContents error: %v", err)
			}
			modifiedBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read modified body: %v", err)
			}
			if !bytes.Equal(modifiedBody, tt.expected) {
				if len(modifiedBody) < 1024 && len(tt.expected) < 1024 {
					t.Errorf("expected response body %q, got %q", tt.expected, modifiedBody)
				} else {
					t.Error("expected body != modifiedBody")
				}
			}
		})
	}
}

func newHTTPResponse(body []byte) *http.Response {
	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader(body)),
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
	}
}

func genAlphanumByteArray(length int) []byte {
	const alphanumerics = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	alphanumLen := len(alphanumerics)
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = alphanumerics[rand.Intn(alphanumLen)] // #nosec G404 -- not used for security purposes
	}

	return result
}
