package networkrules

import (
	"net/http"
)

// CreateBlockResponse creates a response for a blocked request.
func (nr *NetworkRules) CreateBlockResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusForbidden,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Proto:      req.Proto,
	}
}

// CreateRedirectResponse creates a response for a redirected request.
func (nr *NetworkRules) CreateRedirectResponse(req *http.Request, to string) *http.Response {
	return &http.Response{
		// The use of 307 Temporary Redirect instead of 308 Permanent Redirect is intentional.
		// 308's can be cached by clients, which might cause issues in cases of erroneous redirects, changing filter rules, etc.
		StatusCode: http.StatusTemporaryRedirect,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Proto:      req.Proto,
		Header: http.Header{
			"Location": []string{to},
		},
	}
}
