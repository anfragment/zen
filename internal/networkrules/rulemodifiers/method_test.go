package rulemodifiers

import (
	"net/http"
	"testing"
)

func TestMethodModifier(t *testing.T) {
	t.Parallel()

	t.Run("single method", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		m.Parse("method=GET")
		req := http.Request{
			Method: "GET",
		}
		if !m.ShouldMatchReq(&req) {
			t.Error("method=GET should match a GET request")
		}
	})

	t.Run("single inverted method", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		m.Parse("method=~GET")
		req := http.Request{
			Method: "GET",
		}
		if m.ShouldMatchReq(&req) {
			t.Error("method=~GET should not match a GET request")
		}
	})

	t.Run("lowercase method", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		m.Parse("method=get")
		req := http.Request{
			Method: "GET",
		}
		if !m.ShouldMatchReq(&req) {
			t.Error("method=get should match a GET request")
		}
	})

	t.Run("multiple methods", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		m.Parse("method=GET|POST")

		req := http.Request{
			Method: "GET",
		}
		if !m.ShouldMatchReq(&req) {
			t.Error("method=GET|POST should match a GET request")
		}

		req.Method = "POST"
		if !m.ShouldMatchReq(&req) {
			t.Error("method=GET|POST should match a POST request")
		}

		req.Method = "HEAD"
		if m.ShouldMatchReq(&req) {
			t.Error("method=GET|POST should not match a HEAD request")
		}
	})

	t.Run("multiple inverted methods", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		m.Parse("method=~GET|~POST")

		req := http.Request{
			Method: "GET",
		}
		if m.ShouldMatchReq(&req) {
			t.Error("method=~GET|~POST should not match a GET request")
		}

		req.Method = "POST"
		if m.ShouldMatchReq(&req) {
			t.Error("method=~GET|~POST should not match a POST request")
		}

		req.Method = "HEAD"
		if !m.ShouldMatchReq(&req) {
			t.Error("method=~GET|~POST should match a HEAD request")
		}

		req.Method = "PUT"
		if !m.ShouldMatchReq(&req) {
			t.Error("method=~GET|~POST should match a PUT request")
		}
	})

	t.Run("mixed inverted and non-inverted methods", func(t *testing.T) {
		t.Parallel()

		m := MethodModifier{}
		if err := m.Parse("method=GET|~POST"); err == nil {
			t.Error("method=GET|~POST should return an error")
		}

		m = MethodModifier{}
		if err := m.Parse("method=~GET|POST"); err == nil {
			t.Error("method=~GET|POST should return an error")
		}
	})

	t.Run("cancels", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			a        MethodModifier
			b        MethodModifier
			expected bool
		}{
			{
				name: "Should cancel - identical modifiers",
				a: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				expected: true,
			},
			{
				name:     "Should cancel - identical modifiers",
				a:        MethodModifier{},
				b:        MethodModifier{},
				expected: true,
			},
			{
				name: "Should cancel - identical methods but different order",
				a: MethodModifier{
					entries: []methodModifierEntry{
						{method: "POST"},
						{method: "GET"},
					},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				expected: true,
			},
			{
				name: "Should not cancel - different method entries",
				a: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "PUT"},
						{method: "DELETE"},
					},
					inverted: true,
				},
				expected: false,
			},
			{
				name: "Should not cancel - different inverted flag",
				a: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: false,
				},
				expected: false,
			},
			{
				name: "Should not cancel - one has extra method",
				a: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
					},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
						{method: "POST"},
						{method: "DELETE"},
					},
					inverted: true,
				},
				expected: false,
			},
			{
				name: "Should not cancel - one is empty",
				a: MethodModifier{
					entries:  []methodModifierEntry{},
					inverted: true,
				},
				b: MethodModifier{
					entries: []methodModifierEntry{
						{method: "GET"},
					},
					inverted: true,
				},
				expected: false,
			},
			{
				name: "Should cancel - both are empty",
				a: MethodModifier{
					entries:  []methodModifierEntry{},
					inverted: true,
				},
				b: MethodModifier{
					entries:  []methodModifierEntry{},
					inverted: true,
				},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result := tt.a.Cancels(&tt.b)
				if result != tt.expected {
					t.Errorf("MethodModifier.Cancels() = %t, want %t", result, tt.expected)
				}
			})
		}
	})
}
