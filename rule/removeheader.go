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

type removeHeaderModifier struct {
	kind       removeHeaderKind
	headerName string
}

var _ modifyingModifier = (*removeHeaderModifier)(nil)

func (rm *removeHeaderModifier) Parse(modifier string) error {
	if !strings.HasPrefix(modifier, "removeheader=") {
		return errors.New("invalid removeheader modifier")
	}
	modifier = strings.TrimPrefix(modifier, "removeheader=")

	if strings.HasPrefix(modifier, "request:") {
		rm.kind = removeHeaderKindRequest
		rm.headerName = strings.TrimPrefix(modifier, "request:")
		return nil
	}

	rm.kind = removeHeaderKindResponse
	rm.headerName = modifier
	return nil
}

func (rm *removeHeaderModifier) ModifyReq(req *http.Request) (modified bool) {
	if rm.kind != removeHeaderKindRequest {
		return false
	}
	log.Println("RemoveHeaderModifier.ModifyReq", req.Header.Get(rm.headerName), rm.headerName)
	if req.Header.Get(rm.headerName) == "" {
		return false
	}

	req.Header.Del(rm.headerName)
	return true
}

func (rm *removeHeaderModifier) ModifyRes(res *http.Response) (modified bool) {
	if rm.kind != removeHeaderKindResponse {
		return false
	}
	if res.Header.Get(rm.headerName) == "" {
		return false
	}

	res.Header.Del(rm.headerName)
	return true
}
