package rulemodifiers

import (
	"fmt"
	"net/http"
	"strings"
)

// Modifier is a Modifier of a rule.
type Modifier interface {
	Parse(modifier string) error
}

// MatchingModifier defines whether a rule matches a request.
type MatchingModifier interface {
	Modifier
	ShouldMatchReq(req *http.Request) bool
	ShouldMatchRes(res *http.Response) bool
}

// modifyingModifier modifies a request.
type ModifyingModifier interface {
	Modifier
	ModifyReq(req *http.Request) (modified bool)
	ModifyRes(res *http.Response) (modified bool)
}

func ParseModifier(m string) (Modifier, error) {
	if len(m) == 0 {
		return nil, fmt.Errorf("empty modifier")
	}

	isKind := func(kind string) bool {
		if len(m) > 0 && m[0] == '~' {
			return strings.HasPrefix(m[1:], kind)
		}
		return strings.HasPrefix(m, kind)
	}
	var modifier Modifier
	switch {
	case isKind("domain"):
		modifier = &DomainModifier{}
	case isKind("method"):
		modifier = &MethodModifier{}
	case isKind("document"),
		isKind("doc"),
		isKind("xmlhttprequest"),
		isKind("xhr"),
		isKind("font"),
		isKind("subdocument"),
		isKind("image"),
		isKind("object"),
		isKind("script"),
		isKind("stylesheet"),
		isKind("media"),
		isKind("other"):
		modifier = &ContentTypeModifier{}
	case isKind("third-party"):
		modifier = &ThirdPartyModifier{}
	case isKind("removeparam"):
		modifier = &RemoveParamModifier{}
	case isKind("header"):
		modifier = &HeaderModifier{}
	case isKind("removeheader"):
		modifier = &RemoveHeaderModifier{}
	// case isKind("all"):
	// TODO: should act as "popup" modifier once it gets implemented
	default:
		return nil, fmt.Errorf("unknown modifier %s", m)
	}

	if err := modifier.Parse(m); err != nil {
		return nil, err
	}

	return modifier, nil
}
