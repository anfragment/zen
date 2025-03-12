package rulemodifiers

import (
	"net/http"
	"regexp"
	"testing"
)

func TestHeaderModifier(t *testing.T) {
	t.Parallel()

	t.Run("fails on empty modifier", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse(""); err == nil {
			t.Error("headerModifier.Parse(\"\") = nil, want error")
		}
	})

	t.Run("fails on missing specifier", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse("header="); err == nil {
			t.Error("headerModifier.Parse(\"header=\") = nil, want error")
		}
	})

	t.Run("fails on invalid specifier", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse("header=one:two:three"); err == nil {
			t.Error("headerModifier.Parse(\"header=one:two:three\") = nil, want error")
		}
	})

	t.Run("matches response by header name", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse("header=Content-Type"); err != nil {
			t.Fatalf("headerModifier.Parse(\"header=Content-Type\") = %v, want nil", err)
		}

		res := &http.Response{
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		}
		if !m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = false, want true")
		}

		res.Header.Del("Content-Type")
		if m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = true, want false")
		}
	})

	t.Run("matches response by header name and exact value", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse("header=Content-Type:application/json"); err != nil {
			t.Fatalf("headerModifier.Parse(\"header=Content-Type:application/json\") = %v, want nil", err)
		}

		res := &http.Response{
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		}
		if !m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = false, want true")
		}

		res.Header.Set("Content-Type", "application/xml")
		if m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = true, want false")
		}

		res.Header.Del("Content-Type")
		if m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = true, want false")
		}
	})

	t.Run("matches response by header name and regexp value", func(t *testing.T) {
		t.Parallel()
		m := HeaderModifier{}
		if err := m.Parse("header=Content-Type:/application/i"); err != nil {
			t.Fatalf("headerModifier.Parse(\"header=Content-Type:/application/i\") = %v, want nil", err)
		}

		res := &http.Response{
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		}
		if !m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = false, want true")
		}

		res.Header.Set("Content-Type", "application/xml")
		if !m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = false, want true")
		}

		res.Header.Set("Content-Type", "text/plain")
		if m.ShouldMatchRes(res) {
			t.Error("headerModifier.ShouldMatchRes(res) = true, want false")
		}
	})

	t.Run("Cancels", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			a    HeaderModifier
			b    HeaderModifier
			want bool
		}{
			{
				name: "Should cancel - identical modifiers",
				a: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value$"),
				},
				b: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value$"),
				},
				want: true,
			},
			{
				name: "Should cancel - empty",
				a:    HeaderModifier{},
				b:    HeaderModifier{},
				want: true,
			},
			{
				name: "Should not cancel - different header names",
				a: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value$"),
				},
				b: HeaderModifier{
					name:   "X-Different",
					exact:  "value",
					regexp: regexp.MustCompile("^value$"),
				},
				want: false,
			},
			{
				name: "Should not cancel - different exact values",
				a: HeaderModifier{
					name:   "X-Test",
					exact:  "value1",
					regexp: regexp.MustCompile("^value$"),
				},
				b: HeaderModifier{
					name:   "X-Test",
					exact:  "value2",
					regexp: regexp.MustCompile("^value$"),
				},
				want: false,
			},
			{
				name: "Should not cancel - different regex values",
				a: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value1$"),
				},
				b: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value2$"),
				},
				want: false,
			},
			{
				name: "Should not cancel - one regex is nil",
				a: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: nil,
				},
				b: HeaderModifier{
					name:   "X-Test",
					exact:  "value",
					regexp: regexp.MustCompile("^value$"),
				},
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result := tt.a.Cancels(&tt.b)
				if result != tt.want {
					t.Errorf("HeaderModifier.Cancels() = %t, want %t", result, tt.want)
				}
			})
		}
	})
}
