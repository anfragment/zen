package filter

import (
	"reflect"
	"testing"
)

type matchTestCase struct {
	url  string
	want bool
}

type matchTest struct {
	name string
	rule string
	urls []matchTestCase
}

func (mt *matchTest) run(t *testing.T) {
	matcher := NewMatcher()
	matcher.AddRule(mt.rule)
	for _, u := range mt.urls {
		if got := matcher.Match(u.url); got != u.want {
			t.Errorf("%s: Match(%q) = %v, want %v", mt.name, u.url, got, u.want)
		}
	}
}

func TestMatcherByAddressParts(t *testing.T) {
	t.Parallel()
	tests := []matchTest{
		{
			name: "by address parts",
			rule: "/banner/img",
			urls: []matchTestCase{
				{"http://example.com/banner/img", true},
				{"https://example.com/banner/img", true},
				{"http://example.com/example/banner/img", true},
				{"http://example.com/banner/img/example", true},
				{"http://example.com/banner-img", false},
				{"https://example.com/banner?img", false},
				{"http://example.com", false},
				{"", false},
			},
		},
		{
			name: "by segments",
			rule: "-banner-ad-",
			urls: []matchTestCase{
				{"http://example.com/-banner-ad-", true},
				{"https://example.com/-example-banner-ad-example", true},
				{"http://example.com/-banner-ad-example", true},
				{"http://example.com/banner-ad", false},
				{"https://example.com/banner-ad", false},
				{"https://example.com/this-is-a-banner-ad", false},
				{"http://example.com/ad-banner", false},
				{"http://example.com/banner-ad-", false},
				{"https://example.com/-banner-ad", false},
				{"http://example.com/banner-ad-example", false},
				{"http://example.com/banner?ad", false},
				{"", false},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestTokenize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		s      string
		tokens []string
	}{
		// {"", []string{}},
		{"http://example.com", []string{"http", "://", "example", ".", "com"}},
		{"http://example.com/", []string{"http", "://", "example", ".", "com", "/"}},
		{"http://example.com/?q=example", []string{"http", "://", "example", ".", "com", "/", "?", "q", "=", "example"}},
		{"https://example.com/subdir/doc?foo1=bar1&foo2=bar2", []string{"https", "://", "example", ".", "com", "/", "subdir", "/", "doc", "?", "foo1", "=", "bar1", "&", "foo2", "=", "bar2"}},
		{"-banner-ad-", []string{"-", "banner", "-", "ad", "-"}},
		{"banner", []string{"banner"}},
		{"/banner/img", []string{"/", "banner", "/", "img"}},
	}

	for _, test := range tests {
		if got := tokenize(test.s); !reflect.DeepEqual(got, test.tokens) {
			t.Errorf("Tokenize(%q) = %#v, want %#v", test.s, got, test.tokens)
		}
	}
}
