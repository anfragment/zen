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
