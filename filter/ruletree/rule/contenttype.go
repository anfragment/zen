package rule

import (
	"net/http"
)

type contentTypeModifier struct {
	contentType string
	inverted    bool
}

func (m *contentTypeModifier) Parse(modifier string) error {
	if modifier[0] == '~' {
		m.inverted = true
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
		"doc":       "document",
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
