package htmlrewrite

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/html"
)

// PrependHeadContents allows to prepend the contents of the <head> tag in an HTTP text/html response.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func PrependHeadContents(res *http.Response, prependWith []byte) error {
	rawBodyReader, err := getRawBodyReader(res)
	if err != nil {
		return fmt.Errorf("get raw body reader: %w", err)
	}

	reader, writer := io.Pipe()

	go func() {
		defer rawBodyReader.Close()

		z := html.NewTokenizer(rawBodyReader)

	outer:
		for {
			switch token := z.Next(); token {
			case html.ErrorToken:
				writer.CloseWithError(z.Err())
				break outer
			case html.StartTagToken:
				writer.Write(z.Raw())
				if name, _ := z.TagName(); bytes.Equal(name, []byte("head")) {
					writer.Write(prependWith)
					writer.Write(z.Buffered())
					// Directly copy the remaining content, without the overhead of tokenization.
					_, err := io.Copy(writer, rawBodyReader)
					writer.CloseWithError(err)
					break outer
				}
			default:
				writer.Write(z.Raw())
			}
		}
	}()

	setBody(res, reader)
	return nil
}

// PrependBodyContents allows to prepend the contents of the <body> tag in an HTTP text/html response.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func PrependBodyContents(res *http.Response, prependWith []byte) error {
	rawBodyReader, err := getRawBodyReader(res)
	if err != nil {
		return fmt.Errorf("get raw body reader: %w", err)
	}

	reader, writer := io.Pipe()

	go func() {
		defer rawBodyReader.Close()

		z := html.NewTokenizer(rawBodyReader)

	outer:
		for {
			switch token := z.Next(); token {
			case html.ErrorToken:
				writer.CloseWithError(z.Err())
				break outer
			case html.StartTagToken:
				writer.Write(z.Raw())
				if name, _ := z.TagName(); bytes.Equal(name, []byte("body")) {
					writer.Write(prependWith)
					writer.Write(z.Buffered())
					// Directly copy the remaining content, without the overhead of tokenization.
					_, err := io.Copy(writer, rawBodyReader)
					writer.CloseWithError(err)
					break outer
				}
			default:
				writer.Write(z.Raw())
			}
		}
	}()

	setBody(res, reader)
	return nil
}

// AppendHeadContents allows to append the contents of the <head> tag in an HTTP text/html response.
//
// On error, the response body is unchanged and the caller may proceed as if the function had not been called.
func AppendHeadContents(res *http.Response, appendWith []byte) error {
	rawBodyReader, err := getRawBodyReader(res)
	if err != nil {
		return fmt.Errorf("get raw body reader: %w", err)
	}

	reader, writer := io.Pipe()

	go func() {
		defer rawBodyReader.Close()

		z := html.NewTokenizer(rawBodyReader)

	outer:
		for {
			switch token := z.Next(); token {
			case html.ErrorToken:
				writer.CloseWithError(z.Err())
				break outer
			case html.EndTagToken:
				if name, _ := z.TagName(); bytes.Equal(name, []byte("head")) {
					writer.Write(appendWith)
					writer.Write(z.Raw())
					writer.Write(z.Buffered())
					// Directly copy the remaining content, without the overhead of tokenization.
					_, err := io.Copy(writer, rawBodyReader)
					writer.CloseWithError(err)
					break outer
				}
				writer.Write(z.Raw())
			default:
				writer.Write(z.Raw())
			}
		}
	}()

	setBody(res, reader)
	return nil
}

func setBody(res *http.Response, body io.ReadCloser) {
	res.Body = body
	// The resulting Content-Length cannot be determined after modifications.
	// Transmit the response as chunked to allow for HTTP connection reuse without having to TCP FIN terminate the connection.
	res.ContentLength = -1
	res.Header.Del("Content-Length")
	res.Header.Del("Content-Encoding")
	res.TransferEncoding = []string{"chunked"}
	res.Header.Set("Content-Type", "text/html;charset=utf-8")
}
