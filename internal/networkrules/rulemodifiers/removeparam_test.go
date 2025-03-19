package rulemodifiers

import (
	"net/http"
	"regexp"
	"testing"
)

func TestRemoveParamModifier(t *testing.T) {
	t.Parallel()

	t.Run("ModifyReq", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			modifier RemoveParamModifier
			url      string
			want     string
			modified bool
		}{
			{
				name:     "generic removes all params",
				modifier: RemoveParamModifier{kind: removeparamKindGeneric},
				url:      "https://example.com/?id=123&name=test",
				want:     "https://example.com/",
				modified: true,
			},
			{
				name:     "exact removes the specified param",
				modifier: RemoveParamModifier{kind: removeparamKindExact, param: "id"},
				url:      "https://example.com/?id=123&known=1&name=test",
				want:     "https://example.com/?known=1&name=test",
				modified: true,
			},
			{
				name:     "exact leaves the URL unchanged if the param is not found",
				modifier: RemoveParamModifier{kind: removeparamKindExact, param: "unknown"},
				url:      "https://example.com/?id=123&name=test&known=1",
				want:     "https://example.com/?id=123&name=test&known=1",
				modified: false,
			},
			{
				name:     "exact inverse removes all params except the specified one",
				modifier: RemoveParamModifier{kind: removeparamKindExactInverse, param: "id"},
				url:      "https://example.com/?id=123&name=test&known=1",
				want:     "https://example.com/?id=123",
				modified: true,
			},
			{
				name:     "regexp removes matching params",
				modifier: RemoveParamModifier{kind: removeparamKindRegexp, regexp: regexp.MustCompile(`id=\d+`)},
				url:      "https://example.com/?id=123&name=test",
				want:     "https://example.com/?name=test",
				modified: true,
			},
			{
				name:     "regexp leaves the URL unchanged if the param is not found",
				modifier: RemoveParamModifier{kind: removeparamKindRegexp, regexp: regexp.MustCompile(`id=[a-z]+`)},
				url:      "https://example.com/?id=123&name=test",
				want:     "https://example.com/?id=123&name=test",
				modified: false,
			},
			{
				name:     "exact leaves the URL unchanged if the URL has no query params",
				modifier: RemoveParamModifier{kind: removeparamKindGeneric},
				url:      "https://example.com:443/",
				want:     "https://example.com:443/",
				modified: false,
			},
			{
				name:     "for multiple values with the same key, only the matching values are removed",
				modifier: RemoveParamModifier{kind: removeparamKindRegexp, regexp: regexp.MustCompile(`^id=\d+$`)},
				url:      "https://example.com/?id=test0&id=1&id=2&id=3&id=test1&id=test2",
				want:     "https://example.com/?id=test0&id=test1&id=test2",
				modified: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req, err := http.NewRequest("GET", tt.url, nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				modified := tt.modifier.ModifyReq(req)
				if modified != tt.modified {
					t.Errorf("ModifyReq() modified = %v, want %v", modified, tt.modified)
				}

				got := req.URL.String()
				if got != tt.want {
					t.Errorf("ModifyReq() got URL = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("cancels", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			a        RemoveParamModifier
			b        RemoveParamModifier
			expected bool
		}{
			{
				"Should cancel - identical modifiers",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				true,
			},
			{
				"Should cancel - empty",
				RemoveParamModifier{},
				RemoveParamModifier{},
				true,
			},
			{
				"Should not cancel - different param",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				RemoveParamModifier{
					kind:   1,
					param:  "user",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				false,
			},
			{
				"Should not cancel - different kind",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				RemoveParamModifier{
					kind:   2,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				false,
			},
			{
				"Should not cancel - different regex",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile("^[a-zA-Z]+$"),
				},
				false,
			},
			{
				"Should cancel - both regex nil",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				true,
			},
			{
				"Should not cancel - one regex nil, one non-nil",
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result := tt.a.Cancels(&tt.b)
				if result != tt.expected {
					t.Errorf("RemoveParamModifier.Cancels() = %t, want %t", result, tt.expected)
				}
			})
		}
	})
}
