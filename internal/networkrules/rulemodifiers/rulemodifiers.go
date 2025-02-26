package rulemodifiers

import "net/http"

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
