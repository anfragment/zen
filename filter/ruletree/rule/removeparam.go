package rule

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

type removeParamModifier struct {
	kind   removeparamKind
	param  string
	regexp *regexp.Regexp
}

func (rm *removeParamModifier) Parse(modifier string) error {
	if modifier == "removeparam" {
		rm.kind = removeparamKindGeneric
		return nil
	}

	eqIndex := strings.IndexByte(modifier, '=')
	if eqIndex == -1 {
		return fmt.Errorf("invalid removeparam modifier")
	}
	value := modifier[eqIndex+1:]

	if value[0] == '/' && value[len(value)-1] == '/' {
		regexp, err := regexp.Compile(value[1 : len(value)-1])
		if err != nil {
			return fmt.Errorf("invalid regexp %w", value)
		}
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

func (rm *removeParamModifier) Modify(req *http.Request) (modified bool) {
	query := req.URL.Query()
	params := make([]string, len(query))
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
			if rm.regexp.MatchString(param) {
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
