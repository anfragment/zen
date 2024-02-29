package rule

import (
	"net/http"
	"testing"
)

func TestDomainModifierMatching(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		name        string
		modifier    string
		referer     string
		shouldMatch bool
	}{
		{"Single domain - match", "domain=example.com", "http://example.com/", true},
		{"Single domain - subdomain match", "domain=example.com", "http://sub.example.com/path", true},
		{"Single domain - no match", "domain=example.com", "http://example.org/", false},
		{"Single inverted domain - match", "domain=~example.com", "http://test.com/", true},
		{"Single inverted domain - no match", "domain=~example.com", "http://example.com/", false},
		{"TLD - match", "domain=example.*", "http://example.com/", true},
		{"TLD - subdomain and path match", "domain=example.*", "https://example.co.uk/some/path", true},
		{"TLD - no match", "domain=example.*", "http://test.com", false},
		{"Regex - match com", `domain=/^example\.(com|org)$/`, "http://example.com/", true},
		{"Regex - match org", `domain=/^example\.(com|org)$/`, "http://example.org/", true},
		{"Regex - no match", `domain=/^example\.(com|org)$/`, "http://example.net/", false},
		{"Multiple domains - match com", "domain=example.com|example.org", "http://example.com/", true},
		{"Multiple domains - match org", "domain=example.com|example.org", "http://example.org/", true},
		{"Multiple domains - no match", "domain=example.com|example.org", "http://example.net/", false},
		{"Multiple inverted domains - match", "domain=~example.com|~example.org", "http://example.net/", true},
		{"Multiple inverted domains - no match com", "domain=~example.com|~example.org", "http://example.com/", false},
		{"Multiple inverted domains - no match org", "domain=~example.com|~example.org", "http://example.org/", false},
	}

	for _, tt := range tests {
		tt := tt // TODO: remove when updating to Go 1.22
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newDomainModifier(t, tt.modifier)
			req := newRequestWithReferer(tt.referer)
			if got := m.ShouldMatch(req); got != tt.shouldMatch {
				t.Errorf("domainModifier{%s}.ShouldMatch(%s) = %v, want %v", tt.modifier, tt.referer, got, tt.shouldMatch)
			}
		})
	}
}

func TestDomainModifierParse(t *testing.T) {
	t.Parallel()

	t.Run("Should fail on empty modifier", func(t *testing.T) {
		t.Parallel()
		m := domainModifier{}
		if err := m.Parse("domain="); err == nil {
			t.Error("domainModifier.Parse(\"domain=\") = nil, want error")
		}
	})

	t.Run("Should fail on inverted and non-inverted domains", func(t *testing.T) {
		t.Parallel()
		m := domainModifier{}
		if err := m.Parse("domain=example.com|~example.org"); err == nil {
			t.Error("domainModifier.Parse(\"domain=example.com|~example.org\") = nil, want error")
		}
	})
}

func newDomainModifier(t *testing.T, domain string) domainModifier {
	t.Helper()
	m := domainModifier{}
	if err := m.Parse(domain); err != nil {
		t.Fatal(err)
	}
	return m
}

func newRequestWithReferer(referer string) *http.Request {
	return &http.Request{
		Header: http.Header{
			"Referer": []string{referer},
		},
	}
}
