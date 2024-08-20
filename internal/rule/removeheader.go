package rule

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

type removeHeaderKind int8

const (
	removeHeaderKindResponse removeHeaderKind = iota
	removeHeaderKindRequest
)

var forbiddenHeaders = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Credentials",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Methods",
	"Access-Control-Expose-Headers",
	"Access-Control-Max-Age",
	"Access-Control-Request-Headers",
	"Access-Control-Request-Method",
	"Origin",
	"Timing-Allow-Origin",
	"Allow",
	"Cross-Origin-Embedder-Policy",
	"Cross-Origin-Opener-Policy",
	"Cross-Origin-Resource-Policy",
	"Content-Security-Policy",
	"Content-Security-Policy-Report-Only",
	"Expect-CT",
	"Feature-Policy",
	"Permissions-Policy",
	"Origin-Isolation",
	"Strict-Transport-Security",
	"Upgrade-Insecure-Requests",
	"X-Content-Type-Options",
	"X-Download-Options",
	"X-Frame-Options",
	"X-Permitted-Cross-Domain-Policies",
	"X-Powered-By",
	"X-XSS-Protection",
	"Public-Key-Pins",
	"Public-Key-Pins-Report-Only",
	"Sec-WebSocket-Key",
	"Sec-WebSocket-Extensions",
	"Sec-WebSocket-Accept",
	"Sec-WebSocket-Protocol",
	"Sec-WebSocket-Version",
	"Sec-Fetch-Mode",
	"Sec-Fetch-Dest",
	"Sec-Fetch-Site",
	"Sec-Fetch-User",
	"Referrer-Policy",
	"Content-Type",
	"Content-Length",
	"Accept",
	"Accept-Encoding",
	"Host",
	"Connection",
	"Transfer-Encoding",
	"Upgrade",
	"P3P",
}

var (
	ErrForbiddenHeader             = errors.New("forbidden header")
	ErrInvalidRemoveheaderModifier = errors.New("invalid removeheader modifier")
)

type removeHeaderModifier struct {
	kind       removeHeaderKind
	headerName string
}

var _ modifyingModifier = (*removeHeaderModifier)(nil)

func (rm *removeHeaderModifier) Parse(modifier string) error {
	if !strings.HasPrefix(modifier, "removeheader=") {
		return ErrInvalidRemoveheaderModifier
	}
	modifier = strings.TrimPrefix(modifier, "removeheader=")

	switch {
	case strings.HasPrefix(modifier, "request:"):
		rm.kind = removeHeaderKindRequest
		rm.headerName = strings.TrimPrefix(modifier, "request:")
	default:
		rm.kind = removeHeaderKindResponse
		rm.headerName = modifier
	}

	rm.headerName = http.CanonicalHeaderKey(rm.headerName)

	for _, forbiddenHeader := range forbiddenHeaders {
		if rm.headerName == forbiddenHeader {
			log.Printf("WARNING: FOUND FORBIDDEN $removeheader %s", forbiddenHeader)
			return ErrForbiddenHeader
		}
	}

	return nil
}

func (rm *removeHeaderModifier) ModifyReq(req *http.Request) (modified bool) {
	if rm.kind != removeHeaderKindRequest {
		return false
	}
	// Since rm.headerName is already in canonical form, we can use the map directly instead of the Get/Del API.
	if len(req.Header[rm.headerName]) == 0 {
		return false
	}

	delete(req.Header, rm.headerName)
	return true
}

func (rm *removeHeaderModifier) ModifyRes(res *http.Response) (modified bool) {
	if rm.kind != removeHeaderKindResponse {
		return false
	}
	// Since rm.headerName is already in canonical form, we can use the map directly instead of the Get/Del API.
	if len(res.Header[rm.headerName]) == 0 {
		return false
	}

	delete(res.Header, rm.headerName)
	return true
}
