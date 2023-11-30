package config

import (
	"context"
	"log"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is the current Version of the binary. Set at compile time using ldflags (see .github/workflows/build.yml)
const Version = "development"

func SelfUpdate(ctx context.Context) {
	shouldUpdate, release := checkForUpdates()
	if !shouldUpdate {
		return
	}

	action, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Title:         "Would you like to update Zen?",
		Message:       release.ReleaseNotes,
		Buttons:       []string{"Yes", "No"},
		Type:          runtime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		log.Printf("error occurred while showing update dialog: %v", err)
		return
	}
	if action == "No" {
		return
	}

	v := semver.MustParse(Version)
	_, err = selfupdate.UpdateSelf(v, "anfragment/zen")
	if err != nil {
		log.Printf("error occurred while updating binary: %v", err)
		return
	}

	action, err = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Title:         "Zen has been updated",
		Message:       "Zen has been updated to the latest version. Would you like to restart it now?",
		Buttons:       []string{"Yes", "No"},
		Type:          runtime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		log.Printf("error occurred while showing restart dialog: %v", err)
	}
	if action == "Yes" {
		runtime.WindowReloadApp(ctx)
	}
}

func checkForUpdates() (bool, *selfupdate.Release) {
	if Version == "development" {
		return false, nil
	}

	latest, found, err := selfupdate.DetectLatest("anfragment/zen")
	if err != nil {
		log.Printf("error occurred while detecting version: %v", err)
		return false, nil
	}

	v := semver.MustParse(Version)
	if !found || latest.Version.LTE(v) {
		return false, nil
	}

	return true, latest
}
