package rulemodifiers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type removeparamKind int8

const (
	removeparamKindGeneric removeparamKind = iota
	removeparamKindRegexp
	removeparamKindExact
	removeparamKindExactInverse
)

type RemoveParamModifier struct {
	kind   removeparamKind
	param  string
	regexp *regexp.Regexp
}

var _ ModifyingModifier = (*RemoveParamModifier)(nil)

func (rm *RemoveParamModifier) Parse(modifier string) error {
	if modifier == "removeparam" {
		rm.kind = removeparamKindGeneric
		return nil
	}

	eqIndex := strings.IndexByte(modifier, '=')
	if eqIndex == -1 {
		return fmt.Errorf("invalid removeparam modifier")
	}
	value := modifier[eqIndex+1:]

	regexp, err := parseRegexp(value)
	if err != nil {
		return fmt.Errorf("parse regexp: %w", err)
	}
	if regexp != nil {
		rm.kind = removeparamKindRegexp
		rm.regexp = regexp
		return nil
	}

	if value[0] == '~' {
		rm.kind = removeparamKindExactInverse
		rm.param = value[1:]
		return nil
	}

	rm.kind = removeparamKindExact
	rm.param = value
	return nil
}

func (rm *RemoveParamModifier) ModifyReq(req *http.Request) (modified bool) {
	query := req.URL.Query()
	params := make([]string, 0, len(query))
	for param := range query {
		params = append(params, param)
	}

	switch rm.kind {
	case removeparamKindGeneric:
		for _, param := range params {
			query.Del(param)
			modified = true
		}
	case removeparamKindRegexp:
		for _, param := range params {
			// The second condition addresses an issue with how some filter lists implement regexp removeparam modifiers.
			// For example, here's a rule from DandelionSprout's CleanURLs list:
			// $removeparam=/^utm(_[a-z_]*)?=/
			// The '=' sign at the end would prevent matching the parameter name. Therefore, we check for it separately.
			if rm.regexp.MatchString(param) || rm.regexp.MatchString(param+"=") {
				query.Del(param)
				modified = true
			}
		}
	case removeparamKindExact:
		for _, param := range params {
			if param == rm.param {
				query.Del(param)
				modified = true
			}
		}
	case removeparamKindExactInverse:
		for _, param := range params {
			if param != rm.param {
				query.Del(param)
				modified = true
			}
		}
	}

	if modified {
		req.URL.RawQuery = query.Encode()
	}
	return modified
}

func (rm *RemoveParamModifier) ModifyRes(*http.Response) (modified bool) {
	return false
}

func (rme *RemoveParamModifier) Cancels(m Modifier) bool {
	rm, ok := m.(*RemoveParamModifier)
	if !ok {
		return false
	}

	return rm.kind == rme.kind && rm.param == rme.param && rme.regexp == rm.regexp
}

func (rm *RemoveParamModifier) ShouldMatchReq(req *http.Request) bool  { return false }
func (rm *RemoveParamModifier) ShouldMatchRes(res *http.Response) bool { return false }
