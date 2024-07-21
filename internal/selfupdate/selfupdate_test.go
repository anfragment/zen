package selfupdate

import "testing"

func TestIsNewer(t *testing.T) {
	t.Parallel()

	t.Run("fails on invalid current version", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v100",
		}
		if _, err := su.isNewer("v1.0.0"); err == nil {
			t.Error("got nil, want error")
		}
	})

	t.Run("fails on invalid new version", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v1.0.0",
		}
		if _, err := su.isNewer("v100"); err == nil {
			t.Error("got nil, want error")
		}
	})

	t.Run("returns true if new version has larger patch version than current", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v3.2.2",
		}
		newer, err := su.isNewer("v3.2.3")
		if err != nil {
			t.Error("got error, want nil")
		}
		if !newer {
			t.Error("got false, want true")
		}
	})

	t.Run("returns true if new version has larger minor version than current", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v0.1.0",
		}
		newer, err := su.isNewer("v0.5.0")
		if err != nil {
			t.Error("got error, want nil")
		}
		if !newer {
			t.Error("got false, want true")
		}
	})

	t.Run("returns true if new version has larger major version than current", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v10.0.0",
		}
		newer, err := su.isNewer("v14.2.8")
		if err != nil {
			t.Error("got error, want nil")
		}
		if !newer {
			t.Error("got false, want true")
		}
	})

	t.Run("returns false if versions are equal", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v4.1.1",
		}
		newer, err := su.isNewer("v4.1.1")
		if err != nil {
			t.Error("got error, want nil")
		}
		if newer {
			t.Error("got true, want false")
		}
	})

	t.Run("returns false if new version is older than current", func(t *testing.T) {
		t.Parallel()
		su := SelfUpdater{
			version: "v1.0.0",
		}
		newer, err := su.isNewer("v0.9.9")
		if err != nil {
			t.Error("got error, want nil")
		}
		if newer {
			t.Error("got true, want false")
		}
	})
}
