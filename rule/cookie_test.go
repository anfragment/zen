package rule

import "testing"

func TestCookieModifier(t *testing.T) {
	t.Parallel()

	t.Run("returns error if input is invalid", func(t *testing.T) {
		t.Parallel()

		rm := &cookieModifier{}
		if err := rm.Parse("notcookie"); err == nil {
			t.Error("expected error to be non-nil")
		}
	})

	t.Run("removes request cookie", func(t *testing.T) {
		t.Parallel()

		rm := &cookieModifier{}
		if err := rm.Parse("cookie=")
	})
}
