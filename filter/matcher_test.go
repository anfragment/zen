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
	name  string
	rules []string
	urls  []matchTestCase
}

func (mt *matchTest) run(t *testing.T) {
	matcher := NewMatcher()
	for _, r := range mt.rules {
		matcher.AddRule(r)
	}
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
			name:  "by address parts",
			rules: []string{"/banner/img"},
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
			name:  "by segments",
			rules: []string{"-banner-ad-"},
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
		{
			name: "by multiple segments",
			rules: []string{
				"-banner-ad-",
				"-ad-banner-",
				"-adfliction/",
				"-adframe.",
				".html?clicktag=",
				".html?ad=",
				".html?ad_",
				"/ad-top-",
			},
			urls: []matchTestCase{
				{"http://example.com/-banner-ad-", true},
				{"https://example.com/-ad-banner-", true},
				{"http://example.com/-adfliction/", true},
				{"http://example.com/-adframe.", true},
				{"http://example.com/-adframe.html", true},
				{"http://example.com/innocent.html?clicktag=", true},
				{"http://example.com/innocent.html?ad=", true},
				{"http://example.com/innocent.html?ad_", true},
				{"http://example.com/ad-top-", true},
				{"http://example.com/-banner-ad-example", true},
				{"http://example.com/banner-ad", false},
				{"https://example.com/banner-ad", false},
				{"http://test.org", false},
				{"https://example.com/this-is-a-banner-ad", false},
				{"http://example.com/ad-banner", false},
				{"http://example.com/test.html?click=", false},
				{"http://example.com/test.html?ad", false},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestMatcherHosts(t *testing.T) {
	t.Parallel()
	tests := []matchTest{
		{
			name:  "single host",
			rules: []string{"0.0.0.0 example.com"},
			urls: []matchTestCase{
				{"http://example.com", true},
				{"https://example.com", true},
				{"http://example.com/", true},
				{"http://example.com/?q=example", true},
				{"https://example.com/subdir/doc?foo1=bar1&foo2=bar2", true},
				{"http://example.com:8080", true},
				{"https://example.co", false},
				{"http://example.co", false},
				{"http://example.com.co", false},
				{"http://example.com.co/", false},
				{"http://example.com.co/?q=example", false},
			},
		},
		{
			name:  "multiple components",
			rules: []string{"0.0.0.0 sub.test.example.com"},
			urls: []matchTestCase{
				{"http://sub.test.example.com", true},
				{"https://sub.test.example.com", true},
				{"http://sub.test.example.com/", true},
				{"http://sub.test.example.com/?q=example", true},
				{"https://sub.test.example.com/subdir/doc?foo1=bar1&foo2=bar2", true},
				{"http://sub.test.example.com:8080", true},
				{"https://test.example.com", false},
				{"http://test.example.com", false},
				{"http://sub.test.example.co", false},
				{"http://sub.test.example.co/", false},
				{"http://sub.test.example.co/?q=example", false},
			},
		},
		{
			name: "multiple hosts",
			rules: []string{
				"0.0.0.0 example.com",
				"127.0.0.1 example.org",
				"0.0.0.0 test.sub.foo.xyz",
			},
			urls: []matchTestCase{
				{"http://example.com", true},
				{"https://example.com", true},
				{"http://example.org", true},
				{"https://example.org", true},
				{"http://test.sub.foo.xyz", true},
				{"https://test.sub.foo.xyz", true},
				{"http://example.com/", true},
				{"https://example.com/", true},
				{"http://example.com/?q=example", true},
				{"https://example.com/?q=example", true},
				{"https://example.com/subdir/doc?foo1=bar1&foo2=bar2", true},
				{"http://example.com:8080", true},
				{"https://example.co", false},
				{"http://test.sub.foo", false},
				{"http://example.edu", false},
				{"http://example.edu/doc", false},
			},
		},
		{
			name: "multiple overlapping hosts",
			rules: []string{
				"0.0.0.0 example.com",
				"0.0.0.0 example.com.co",
			},
			urls: []matchTestCase{
				{"http://example.com", true},
				{"http://example.com.co", true},
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
		{"example.com", []string{"example", ".", "com"}},
	}

	for _, test := range tests {
		if got := tokenize(test.s); !reflect.DeepEqual(got, test.tokens) {
			t.Errorf("Tokenize(%q) = %#v, want %#v", test.s, got, test.tokens)
		}
	}
}
