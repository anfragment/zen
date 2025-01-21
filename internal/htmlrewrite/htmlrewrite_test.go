package htmlrewrite_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"

	"github.com/anfragment/zen/internal/htmlrewrite"
)

func TestReplaceBodyContentsPublic(t *testing.T) {
	t.Parallel()

	t.Run("prepends <body> contents", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><head>Test</head><body>Original Body Content</body></html>`)
		expectedBody := []byte(`<html><head>Test</head><body>Non-Original Body Content</body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.PrependBodyContents(res, []byte("Non-")); err != nil {
			t.Fatalf("PrependBodyContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("prepends large <body> contents", func(t *testing.T) {
		t.Parallel()

		contents, err := genAlphanumByteArray(10 * 1024 * 1024) // 10MB
		if err != nil {
			t.Fatalf("generate contents: %v", err)
		}

		originalBody := bytes.Join([][]byte{[]byte(`<html><body>`), contents, []byte(`</body></html>`)}, nil)
		expectedBody := bytes.Join([][]byte{[]byte(`<html><body>Prepended`), contents, []byte(`</body></html>`)}, nil)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.PrependBodyContents(res, []byte("Prepended")); err != nil {
			t.Fatalf("PrependBodyContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Error("modifiedBody != expectedBody")
		}
	})

	t.Run("prepends <body> contents with empty byte array", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><body>Original Body Content</body></html>`)
		expectedBody := []byte(`<html><body>Original Body Content</body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.PrependBodyContents(res, []byte("")); err != nil {
			t.Fatalf("PrependBodyContents error: %v", err)
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

		if err := htmlrewrite.PrependBodyContents(res, []byte("test")); err != nil {
			t.Fatalf("PrependBodyContents error: %v", err)
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

func TestAppendHeadContentsPublic(t *testing.T) {
	t.Parallel()

	t.Run("appends <head> contents", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><head>Original Head Content</head><body>Test</body></html>`)
		expectedBody := []byte(`<html><head>Original Head ContentContent</head><body>Test</body></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.AppendHeadContents(res, []byte("Content")); err != nil {
			t.Fatalf("AppendHeadContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Errorf("expected response body %q, got %q", expectedBody, modifiedBody)
		}
	})

	t.Run("appends large <head> contents", func(t *testing.T) {
		t.Parallel()

		contents, err := genAlphanumByteArray(10 * 1024 * 1024) // 10MB
		if err != nil {
			t.Fatalf("generate contents: %v", err)
		}

		originalBody := bytes.Join([][]byte{[]byte(`<html><head>`), contents, []byte(`</head></html>`)}, nil)
		expectedBody := bytes.Join([][]byte{[]byte(`<html><head>`), contents, []byte(`Appended</head></html>`)}, nil)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.AppendHeadContents(res, []byte("Appended")); err != nil {
			t.Fatalf("AppendHeadContents error: %v", err)
		}

		modifiedBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("failed to read modified body: %v", err)
		}

		if !bytes.Equal(modifiedBody, expectedBody) {
			t.Error("modifiedBody != expectedBody")
		}
	})

	t.Run("prepends <head> contents with empty byte array", func(t *testing.T) {
		t.Parallel()

		originalBody := []byte(`<html><head>Original Head Content</head></html>`)
		expectedBody := []byte(`<html><head>Original Head Content</head></html>`)

		res := newHTTPResponse(originalBody)

		if err := htmlrewrite.AppendHeadContents(res, []byte("")); err != nil {
			t.Fatalf("AppendHeadContents error: %v", err)
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

		if err := htmlrewrite.AppendHeadContents(res, []byte("test")); err != nil {
			t.Fatalf("AppendHeadContents error: %v", err)
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

func genAlphanumByteArray(length int) ([]byte, error) {
	const alphanumerics = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	alphanumLen := len(alphanumerics)
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(alphanumLen)))
		if err != nil {
			return nil, fmt.Errorf("gen random index: %w", err)
		}
		result[i] = alphanumerics[randomIndex.Int64()]
	}

	return result, nil
}
