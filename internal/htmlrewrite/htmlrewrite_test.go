package htmlrewrite_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/anfragment/zen/internal/htmlrewrite"
)

func TestReplaceBodyContentsPublic(t *testing.T) {
	t.Parallel()

	t.Run("replaces <body> contents", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><body>Original Body Content</body></html>`)
		expectedBody := []byte(`<html><body>Modified Body Content</body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceBodyContents(res, func(_ []byte) []byte {
			return []byte("Modified Body Content")
		}); err != nil {
			t.Fatalf("ReplaceBodyContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("replaces <body> contents with empty string", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><body>Original Body Content</body></html>`)
		expectedBody := []byte(`<html><body></body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceBodyContents(res, func(_ []byte) []byte {
			return []byte("")
		}); err != nil {
			t.Fatalf("ReplaceBodyContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("doesn't modify response if no <body> is present", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html></html>`)
		expectedBody := []byte(`<html></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceBodyContents(res, func(_ []byte) []byte {
			return []byte("")
		}); err != nil {
			t.Fatalf("ReplaceBodyContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})
}

func TestReplaceHeadContentsPublic(t *testing.T) {
	t.Parallel()

	t.Run("replaces <head> contents", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><head>Original Head Content</head></html>`)
		expectedBody := []byte(`<html><head>Modified Head Content</head></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceHeadContents(res, func(_ []byte) []byte {
			return []byte("Modified Head Content")
		}); err != nil {
			t.Fatalf("ReplaceHeadContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("replaces <head> contents with empty body", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><head>Original Head Content</head></html>`)
		expectedBody := []byte(`<html><head></head></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceHeadContents(res, func(_ []byte) []byte {
			return []byte("")
		}); err != nil {
			t.Fatalf("ReplaceHeadContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("doesn't modify response if no <head> is present", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><body>Test</body></html>`)
		expectedBody := []byte(`<html><body>Test</body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.ReplaceHeadContents(res, func(_ []byte) []byte {
			return []byte("")
		}); err != nil {
			t.Fatalf("ReplaceHeadContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})
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
