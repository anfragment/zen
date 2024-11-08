package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is the current version of the application. Set at compile time for production builds using ldflags (see tasks in the /tasks/build directory).
var Version = "development"

// noSelfUpdate is set to "true" for builds distributed to package managers to prevent auto-updating. It is typed as a string because the linker allows only setting string variables at compile time (see https://pkg.go.dev/cmd/link).
// Set at compile time using ldflags (see the prod-noupdate task in the /tasks/build directory).
var noSelfUpdate = "false"

// releaseTrack is the release track to follow for updates. It currently only takes the value "stable".
var releaseTrack = "stable"

// manifestsBaseURL is the base URL for fetching update manifests.
const manifestsBaseURL = "https://zenprivacy.net/update-manifests"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type SelfUpdater struct {
	version      string
	noSelfUpdate bool
	releaseTrack string
	httpClient   HTTPClient
}

type Release struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	AssetURL    string `json:"assetURL"`
	SHA256      string `json:"sha256"`
}

func NewSelfUpdater(httpClient HTTPClient) (*SelfUpdater, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient is nil")
	}

	u := SelfUpdater{
		version:      Version,
		releaseTrack: releaseTrack,
		httpClient:   httpClient,
	}
	switch noSelfUpdate {
	case "true":
		u.noSelfUpdate = true
	case "false":
	default:
		return nil, fmt.Errorf("invalid noSelfUpdate value: %s", noSelfUpdate)
	}

	return &u, nil
}

func (su *SelfUpdater) checkForUpdates() (*Release, error) {
	log.Println("checking for updates")
	if su.noSelfUpdate {
		log.Println("noSelfUpdate=true, self-update disabled")
		return nil, nil
	}
	if su.version == "development" {
		log.Println("version=development, self-update disabled")
		return nil, nil
	}

	url := fmt.Sprintf("%s/%s/%s/%s/manifest.json", manifestsBaseURL, su.releaseTrack, runtime.GOOS, runtime.GOARCH)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "zen-desktop")

	res, err := su.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	defer res.Body.Close()

	var rel Release
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &rel, nil
}

// isNewer compares the current version with the version passed as an argument and returns true if the argument is newer.
//
// It assumes that both versions are in the format "v<major>.<minor>.<patch>" and returns an error if they are not.
func (su *SelfUpdater) isNewer(version string) (bool, error) {
	var currentMajor, currentMinor, currentPatch, newMajor, newMinor, newPatch int
	if _, err := fmt.Sscanf(su.version, "v%d.%d.%d", &currentMajor, &currentMinor, &currentPatch); err != nil {
		return false, fmt.Errorf("parse current version (%s): %w", su.version, err)
	}
	if _, err := fmt.Sscanf(version, "v%d.%d.%d", &newMajor, &newMinor, &newPatch); err != nil {
		return false, fmt.Errorf("parse new version (%s): %w", version, err)
	}

	if newMajor > currentMajor {
		return true, nil
	}
	if newMajor == currentMajor && newMinor > currentMinor {
		return true, nil
	}
	if newMajor == currentMajor && newMinor == currentMinor && newPatch > currentPatch {
		return true, nil
	}

	return false, nil
}

func (su *SelfUpdater) ApplyUpdate(ctx context.Context) error {
	rel, err := su.checkForUpdates()
	if err != nil {
		return err
	}

	if isNewer, err := su.isNewer(rel.Version); err != nil {
		return err
	} else if !isNewer {
		return nil
	}

	action, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Title:         "Would you like to update Zen?",
		Message:       rel.Description,
		Buttons:       []string{"Yes", "No"},
		Type:          wailsruntime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		log.Printf("error occurred while showing update dialog: %v", err)
		return err
	}
	if action == "No" {
		log.Printf("aborting update, user declined")
		return nil
	}

	ext := filepath.Ext(rel.AssetURL)
	if strings.HasSuffix(rel.AssetURL, ".tar.gz") {
		ext = ".tar.gz"
	}

	if ext != ".tar.gz" && ext != ".zip" {
		return fmt.Errorf("unsupported archive format: %s", ext)
	}

	tmpFile, err := os.CreateTemp("", "downloaded-*"+ext)
	if err != nil {
		return fmt.Errorf("create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = DownloadFile(rel.AssetURL, tmpFile.Name())
	if err != nil {
		return fmt.Errorf("download file: %v", err)
	}

	err = verifyFileHash(tmpFile.Name(), rel.SHA256)
	if err != nil {
		return fmt.Errorf("verify file hash: %v", err)
	}

	var dest string
	switch runtime.GOOS {
	case "darwin":
		dest = "/Applications"
	case "windows":
		dest = os.Getenv("ProgramFiles")
	default:
		panic("unsupported platform")
	}

	err = removeContents(path.Join(dest, "Zen.app"))
	if err != nil {
		return fmt.Errorf("remove contents: %v", err)
	}

	fmt.Println(tmpFile.Name(), dest)

	err = Unarchive(tmpFile.Name(), dest)
	if err != nil {
		return fmt.Errorf("unzip file: %v", err)
	}

	action, err = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Title:         "Zen has been updated",
		Message:       "Zen has been updated to the latest version. Would you like to restart it now?",
		Buttons:       []string{"Yes", "No"},
		Type:          wailsruntime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		log.Printf("error occurred while showing restart dialog: %v", err)
	}
	if action == "Yes" {
		cmd := exec.Command(os.Args[0], os.Args[1:]...) // #nosec G204
		if err := cmd.Start(); err != nil {
			log.Printf("error occurred while restarting: %v", err)
			return err
		}
		wailsruntime.Quit(ctx)
	}

	return nil
}

func DownloadFile(url, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %v", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download file: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write to file: %v", err)
	}

	return nil
}

func verifyFileHash(filePath, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for hashing: %v", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash file: %v", err)
	}

	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, calculatedHash)
	}

	return nil
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
