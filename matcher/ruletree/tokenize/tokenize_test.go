package tokenize

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		s      string
		tokens []string
	}{
		{"", []string{}},
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
