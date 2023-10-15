package rule

import (
	"net/http"
	"net/url"
)

type removeparamModifier struct {
	param string
}

func (m *removeparamModifier) Parse(modifier string) error {
	m.param = modifier
	return nil
}

func (m *removeparamModifier) ShouldMatch(req *http.Request) bool {
	return req.URL.Query().Has(m.param)
}

func (m *removeparamModifier) RedirectTo(req *http.Request) string {
	s := req.URL.String()
	// make a copy of the URL so that we don't modify the original
	// which is used by proxy when forming a response
	url, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	query := url.Query()
	query.Del(m.param)
	url.RawQuery = query.Encode()
	return url.String()
}
