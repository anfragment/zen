package rule

import (
	"net/http"
	"testing"
)

func TestSingleDomain(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	if err := m.Parse("domain=example.com"); err != nil {
		t.Fatal(err)
	}

	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com should match a request with example.com as the referer")
	}

	req.Header.Set("Referer", "http://sub.example.com/path")
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.com should match a request with sub.example.com as the referer")
	}

	req.Header.Set("Referer", "http://example.org/")
	if m.ShouldMatch(&req) {
		t.Error("domain=example.com should not match a request with example.org as the referer")
	}
}

func TestSingleInvertedDomain(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	if err := m.Parse("domain=~example.com"); err != nil {
		t.Fatal(err)
	}

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

	req.Header.Set("Referer", "http://sub.example.com/path")
	if m.ShouldMatch(&req) {
		t.Error("domain=~example.com should not match a request with sub.example.com as the referer")
	}
}

func TestTLD(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	if err := m.Parse("domain=example.*"); err != nil {
		t.Fatal(err)
	}

	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.* should match a request with example.com as the referer")
	}

	req.Header.Set("Referer", "https://example.co.uk/some/path")
	if !m.ShouldMatch(&req) {
		t.Error("domain=example.* should match a request with example.co.uk as the referer")
	}

	req.Header.Set("Referer", "http://test.com")
	if m.ShouldMatch(&req) {
		t.Error("domain=example.* should not match a request with test.com as the referer")
	}
}

func TestRegex(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	if err := m.Parse(`domain=/^example\.(com|org)$/`); err != nil {
		t.Fatal(err)
	}
	req := http.Request{
		Header: http.Header{
			"Referer": []string{"http://example.com/"},
		},
	}
	if !m.ShouldMatch(&req) {
		t.Error(`domain=/^example\.(com|org)$/ should match a request with example.com as the referer`)
	}

	req.Header.Set("Referer", "http://example.org/")
	if !m.ShouldMatch(&req) {
		t.Error(`domain=/^example\.(com|org)$/ should match a request with example.org as the referer`)
	}

	req.Header.Set("Referer", "http://example.net/")
	if m.ShouldMatch(&req) {
		t.Error(`domain=/^example\.(com|org)$/ should not match a request with example.net as the referer`)
	}
}

func TestMultipleDomains(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	if err := m.Parse("domain=example.com|example.org"); err != nil {
		t.Fatal(err)
	}

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
	if err := m.Parse("domain=~example.com|~example.org"); err != nil {
		t.Fatal(err)
	}

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

func TestMixedInvertedAndNonInvertedDomains(t *testing.T) {
	t.Parallel()

	m := domainModifier{}
	err := m.Parse("domain=example.com|~example.org")
	if err == nil {
		t.Error("domain=example.com|~example.org should not be valid")
	}
}
