package rule

import (
	"net/http"
)

type contentTypeModifier struct {
	contentType string
	inverted    bool
}

var _ matchingModifier = (*contentTypeModifier)(nil)

var (
	// secFetchDestMap maps Sec-Fetch-Dest header values to corresponding content type modifiers.
	secFetchDestMap = map[string]string{
		"document": "document",
		"empty":    "xmlhttprequest",
		"font":     "font",
		"frame":    "subdocument",
		"iframe":   "subdocument",
		"image":    "image",
		"object":   "object",
		"script":   "script",
		"style":    "stylesheet",
		"audio":    "media",
		"track":    "media",
		"video":    "media",
	}
	// aliases maps content type aliases to their canonical names.
	aliases = map[string]string{
		"doc": "document",
		"css": "stylesheet",
		"xhr": "xmlhttprequest",
	}
)

func (m *contentTypeModifier) Parse(modifier string) error {
	if modifier[0] == '~' {
		m.inverted = true
		modifier = modifier[1:]
	}
	if canonical, ok := aliases[modifier]; ok {
		modifier = canonical
	}
	m.contentType = modifier
	return nil
}

func (m *contentTypeModifier) ShouldMatchReq(req *http.Request) bool {
	secFetchDest := req.Header.Get("Sec-Fetch-Dest")
	if secFetchDest == "" {
		return false
	}
	contentType, ok := secFetchDestMap[secFetchDest]
	if m.contentType == "other" {
		if m.inverted {
			return ok
		}
		return !ok
	}
	if m.inverted {
		return contentType != m.contentType
	}
	return contentType == m.contentType
}

func (m *contentTypeModifier) ShouldMatchRes(_ *http.Response) bool {
	return false
}
