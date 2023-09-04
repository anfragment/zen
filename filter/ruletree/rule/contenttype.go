package rule

import (
	"net/http"
)

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

var (
	// secFetchDestMap maps the Sec-Fetch-Dest header values to the
	// corresponding content type.
	secFetchDestMap = map[string]string{
		"audio":     "media",
		"document":  "document",
		"empty":     "xmlhttprequest",
		"font":      "font",
		"frame":     "subdocument",
		"iframe":    "subdocument",
		"image":     "image",
		"object":    "object",
		"script":    "script",
		"style":     "stylesheet",
		"track":     "media",
		"video":     "media",
		"websocket": "websocket",
	}
)

func (m *contentTypeModifier) ShouldMatch(req *http.Request) bool {
	contentType, ok := secFetchDestMap[req.Header.Get("Sec-Fetch-Dest")]
	if m.contentType == "other" {
		if m.invert {
			return ok
		}
		return !ok
	}
	if m.invert {
		return contentType != m.contentType
	}
	return contentType == m.contentType
}
