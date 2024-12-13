package htmlrewrite

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

var (
	bodyRegex      = regexp.MustCompile(`(?is)<body[^>]*>.*</body>`)
	bodyStartRegex = regexp.MustCompile(`(?i)<body[^>]*>`)
	bodyEndTagLen  = len("</body>")

	headRegex      = regexp.MustCompile(`(?is)<head[^>]*>.*</head>`)
	headStartRegex = regexp.MustCompile(`(?i)<head[^>]*>`)
	headEndTagLen  = len("</head>")
)

// ReplaceHeadContents allows to replace the contents of the <body> tag in an HTTP response.
// The repl function is called with the contents of the <body> tag and should return the new contents.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func ReplaceBodyContents(res *http.Response, repl func(match []byte) []byte) error {
	rawHTTPBodyBytes, err := readRawBody(res)
	if err != nil {
		return fmt.Errorf("read raw body: %w", err)
	}

	modifiedBody := bodyRegex.ReplaceAllFunc(rawHTTPBodyBytes, func(match []byte) []byte {
		startTagMatches := bodyStartRegex.FindIndex(match)
		if startTagMatches == nil {
			// This check is probably redundant, but let's keep it to avoid a panic in production.
			return nil
		}
		endTagStart := len(match) - bodyEndTagLen

		res := make([]byte, 0, len(match))
		res = append(res, match[:startTagMatches[1]]...)
		res = append(res, repl(match[startTagMatches[1]:endTagStart:endTagStart])...)
		res = append(res, match[endTagStart:]...)
		return res
	})

	setBody(res, modifiedBody)

	return nil
}

// ReplaceHeadContents allows to replace the contents of the <head> tag in an HTTP response.
// The repl function is called with the contents of the <head> tag and should return the new contents.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func ReplaceHeadContents(res *http.Response, repl func(match []byte) []byte) error {
	rawHTTPBodyBytes, err := readRawBody(res)
	if err != nil {
		return fmt.Errorf("read raw body: %w", err)
	}

	modifiedBody := headRegex.ReplaceAllFunc(rawHTTPBodyBytes, func(match []byte) []byte {
		startTagMatches := headStartRegex.FindIndex(match)
		if startTagMatches == nil {
			// This check is probably redundant, but let's keep it to avoid a panic in production.
			return nil
		}
		endTagStart := len(match) - headEndTagLen

		res := make([]byte, 0, len(match))
		res = append(res, match[:startTagMatches[1]]...)
		res = append(res, repl(match[startTagMatches[1]:endTagStart:endTagStart])...)
		res = append(res, match[endTagStart:]...)
		return res
	})

	setBody(res, modifiedBody)

	return nil
}

func setBody(res *http.Response, body []byte) {
	res.Body = io.NopCloser(bytes.NewReader(body))
	res.ContentLength = int64(len(body))
	res.Header.Set("Content-Length", strconv.Itoa(len(body)))
	res.Header.Del("Content-Encoding")
	res.Header.Set("Content-Type", "text/html; charset=utf-8")
}
