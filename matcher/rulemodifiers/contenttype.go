package rulemodifiers

import "net/http"

type contentTypeModifier struct {
	contentType string
	invert      bool
}

func (m *contentTypeModifier) Parse(modifier string) error {
	if modifier[0] == '~' {
		m.invert = true
		modifier = modifier[1:]
	}
	m.contentType = modifier
	return nil
}

func (m *contentTypeModifier) ShouldBlock(req *http.Request) bool {
	dest := req.Header.Get("Sec-Fetch-Dest")
	if dest == "" {
		dest = "document"
	}
	if m.invert {
		return dest != m.contentType
	}
	return dest == m.contentType
}
