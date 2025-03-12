package rulemodifiers

import (
	"regexp"
	"testing"
)

func TestRemoveParamModifier(t *testing.T) {
	t.Parallel()

	t.Run("cancels", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			a        RemoveParamModifier
			b        RemoveParamModifier
			expected bool
		}{
			{
				name: "Should cancel - identical modifiers",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				b: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				expected: true,
			},
			{
				name:     "Should cancel - empty",
				a:        RemoveParamModifier{},
				b:        RemoveParamModifier{},
				expected: true,
			},
			{
				name: "Should not cancel - different param",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				b: RemoveParamModifier{
					kind:   1,
					param:  "user",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				expected: false,
			},
			{
				name: "Should not cancel - different kind",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				b: RemoveParamModifier{
					kind:   2,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				expected: false,
			},
			{
				name: "Should not cancel - different regex",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				b: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile("^[a-zA-Z]+$"),
				},
				expected: false,
			},
			{
				name: "Should cancel - both regex nil",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				b: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				expected: true,
			},
			{
				name: "Should not cancel - one regex nil, one non-nil",
				a: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: nil,
				},
				b: RemoveParamModifier{
					kind:   1,
					param:  "id",
					regexp: regexp.MustCompile(`^\d+$`),
				},
				expected: false,
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
