package filter

import (
	"net/http"

	"github.com/anfragment/zen/filter/ruletree/rule"
)

// formBlockResponse forms a response for a blocked request.
func (m *Filter) formBlockResponse(req *http.Request, rule rule.Rule) *http.Response {
	return &http.Response{
		StatusCode: http.StatusForbidden,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Proto:      req.Proto,
	}
}

// formRedirectResponse forms a response for a redirected request.
func (m *Filter) formRedirectResponse(req *http.Request, to string) *http.Response {
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
