package rule

import (
	"net/http"
	"net/url"
	"testing"
)

func TestSingleDomai(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=example.com")
	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}

	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com should match a request with example.com as the referer")
	}

	req.Header.Set("Referer", "http://example.org/")
	if m.ShouldMatch(&req) {
		t.Error("domain=example.com should not match a request with example.org as the referer")
	}
}

func TestSingleInvertedDomain(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=~example.com")
	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://test.com/"},
		},
	}

	if !m.ShouldMatch(&req) {
		t.Error("domain=~example.com should match a request with test.com as the referer")
	}

	req.Header.Set("Referer", "http://example.com/")
	if m.ShouldMatch(&req) {
		t.Error("domain=~example.com should not match a request with example.com as the referer")
	}
}

func TestMultipleDomains(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=example.com|example.org")

	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com|example.org should match a request with example.com as the referer")
	}

	req.Header.Set("Referer", "http://example.org/")
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com|example.org should match a request with example.org as the referer")
	}

	req.Header.Set("Referer", "http://example.net/")
	if m.ShouldMatch(&req) {
		t.Error("domain=example.com|example.org should not match a request with example.net as the referer")
	}
}

func TestMultipleInvertedDomains(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=~example.com|~example.org")

	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}
	if m.ShouldMatch(&req) {
		t.Error("domain=~example.com|~example.org should not match a request with example.com as the referer")
	}

	req.Header.Set("Referer", "http://example.org/")
	if m.ShouldMatch(&req) {
		t.Error("domain=~example.com|~example.org should not match a request with example.org as the referer")
	}

	req.Header.Set("Referer", "http://example.net/")
	if !m.ShouldMatch(&req) {
		t.Error("domain=~example.com|~example.org should match a request with example.net as the referer")
	}
}

func TestHostnameMatching(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=example.com")

	url, _ := url.Parse("http://example.com/")
	req := http.Request{
		URL:    url,
		Header: http.Header{},
	}

	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com should match a request with example.com as the hostname")
	}

	url, _ = url.Parse("http://example.org/")
	req.URL = url
	if m.ShouldMatch(&req) {
		t.Error("domain=example.com should not match a request with example.org as the hostname")
	}
}

func TestHostnameMatchingWithReferer(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	m.Parse("domain=example.com")

	url, _ := url.Parse("http://example.com/")
	req := http.Request{
		URL: url,
		Header: http.Header{
			"Referer": []string{"http://example.org/"},
		},
	}

	if m.ShouldMatch(&req) {
		t.Error("domain=example.com should not match a request with example.org as the referer")
	}
}

func TestMixedInvertedAndNonInvertedDomains(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	err := m.Parse("domain=example.com|~example.org")
	if err == nil {
		t.Error("domain=example.com|~example.org should not be valid")
	}
}
