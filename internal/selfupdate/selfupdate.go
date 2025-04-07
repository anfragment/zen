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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ZenPrivacy/zen-desktop/internal/cfg"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// NoSelfUpdate is set to "true" for builds distributed to package managers to prevent auto-updating. It is typed as a string because the linker allows only setting string variables at compile time (see https://pkg.go.dev/cmd/link).
// Set at compile time using ldflags (see the prod-noupdate task in the /tasks/build directory).
var NoSelfUpdate = "false"

// releaseTrack is the release track to follow for updates. It currently only takes the value "stable".
var releaseTrack = "stable"

// manifestsBaseURL is the base URL for fetching update manifests.
const manifestsBaseURL = "https://update-manifests.zenprivacy.net"

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type SelfUpdater struct {
	version      string
	noSelfUpdate bool
	policy       cfg.UpdatePolicyType
	releaseTrack string
	httpClient   httpClient
}

type release struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	AssetURL    string `json:"assetURL"`
	SHA256      string `json:"sha256"`
}

const (
	appName = "Zen"
)

func NewSelfUpdater(httpClient httpClient, policy cfg.UpdatePolicyType) (*SelfUpdater, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient is nil")
	}
	if cfg.Version == "" {
		return nil, errors.New("cfg.Version is empty")
	}

	u := SelfUpdater{
		version:      cfg.Version,
		policy:       policy,
		releaseTrack: releaseTrack,
		httpClient:   httpClient,
	}
	switch NoSelfUpdate {
	case "true":
		u.noSelfUpdate = true
	case "false":
	default:
		return nil, fmt.Errorf("invalid noSelfUpdate value: %s", NoSelfUpdate)
	}

	return &u, nil
}

func (su *SelfUpdater) checkForUpdates() (*release, error) {
	log.Println("checking for updates")
	if su.noSelfUpdate {
		log.Println("noSelfUpdate=true, self-update disabled")
		return nil, nil
	}

	if su.policy == cfg.UpdatePolicyDisabled {
		log.Println("updatePolicy=disabled, self-update disabled")
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

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request failed with status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var rel release
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
		return fmt.Errorf("check for updates: %w", err)
	}
	if rel == nil {
		return nil
	}

	if isNewer, err := su.isNewer(rel.Version); err != nil {
		return fmt.Errorf("check if newer: %w", err)
	} else if !isNewer {
		log.Println("no newer version available")
		return nil
	}

	if su.policy == cfg.UpdatePolicyPrompt {
		if proceed, err := su.showUpdateDialog(ctx, rel.Description); err != nil {
			return fmt.Errorf("show update dialog: %w", err)
		} else if !proceed {
			log.Println("aborting update, user declined")
			return nil
		}
	}

	tmpFile, err := su.downloadAndVerifyFile(rel.AssetURL, rel.SHA256)
	if err != nil {
		return fmt.Errorf("download and verify file: %w", err)
	}
	defer os.Remove(tmpFile)

	switch runtime.GOOS {
	case "darwin":
		if err := su.applyUpdateForDarwin(tmpFile); err != nil {
			return fmt.Errorf("apply update: %w", err)
		}
	case "windows", "linux":
		if err := su.applyUpdateForWindowsOrLinux(tmpFile); err != nil {
			return fmt.Errorf("apply update: %w", err)
		}
	default:
		panic("unsupported platform")
	}

	if su.policy == cfg.UpdatePolicyPrompt {
		if restart, err := su.showRestartDialog(ctx); err != nil {
			return fmt.Errorf("show restart dialog: %w", err)
		} else if !restart {
			log.Println("user declined to restart")
			return nil
		}
	}

	if err := su.restartApplication(ctx); err != nil {
		return fmt.Errorf("restart application: %w", err)
	}
	return nil
}

func (su *SelfUpdater) downloadFile(url, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Add("Accept", "application/octet-stream")

	resp, err := su.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download file failed with status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write to file: %w", err)
	}

	return nil
}

func verifyFileHash(filePath, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for hashing: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, calculatedHash)
	}

	return nil
}

func (su *SelfUpdater) showUpdateDialog(ctx context.Context, description string) (bool, error) {
	action, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Title:         "Would you like to update Zen?",
		Message:       description,
		Buttons:       []string{"Yes", "No"},
		Type:          wailsruntime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		return false, fmt.Errorf("show update dialog: %w", err)
	}

	return action == "Yes", nil
}

func (su *SelfUpdater) showRestartDialog(ctx context.Context) (bool, error) {
	action, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Title:         "Zen has been updated",
		Message:       "Zen has been updated to the latest version. Would you like to restart it now?",
		Buttons:       []string{"Yes", "No"},
		Type:          wailsruntime.QuestionDialog,
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	if err != nil {
		return false, fmt.Errorf("show restart dialog: %w", err)
	}
	return action == "Yes", nil
}

func (su *SelfUpdater) downloadAndVerifyFile(assetURL, expectedHash string) (string, error) {
	ext := filepath.Ext(assetURL)
	if strings.HasSuffix(assetURL, ".tar.gz") {
		ext = ".tar.gz"
	}

	if ext != ".tar.gz" && ext != ".zip" {
		return "", fmt.Errorf("unsupported archive format: %s", ext)
	}

	tmpFile, err := os.CreateTemp("", "downloaded-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}

	if err := su.downloadFile(assetURL, tmpFile.Name()); err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}

	if err := verifyFileHash(tmpFile.Name(), expectedHash); err != nil {
		return "", fmt.Errorf("verify file hash: %w", err)
	}

	return tmpFile.Name(), nil
}

func (su *SelfUpdater) applyUpdateForDarwin(tmpFile string) error {
	tempDir, err := os.MkdirTemp("", "unarchive-*")
	if err != nil {
		return fmt.Errorf("create temp unarchive dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := unarchive(tmpFile, tempDir); err != nil {
		return fmt.Errorf("unarchive file: %w", err)
	}

	currentExecPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	appBundlePath := findAppBundlePath(currentExecPath)
	if appBundlePath == "" {
		return fmt.Errorf("application is not running from an .app bundle: %s", currentExecPath)
	}

	newBundlePath, err := findAppBundleInDir(tempDir)
	if err != nil {
		return fmt.Errorf("find new app bundle: %w", err)
	}

	oldBundlePath := generateBackupName(appBundlePath)
	if err := os.Rename(appBundlePath, oldBundlePath); err != nil {
		return fmt.Errorf("rename current app bundle: %w", err)
	}

	rollback := false
	defer func() {
		if rollback {
			log.Printf("Restoring old app bundle from: %s", oldBundlePath)

			if err := os.Rename(oldBundlePath, appBundlePath); err != nil {
				log.Printf("Failed to restore old app bundle: %v", err)
			}
		} else {
			log.Printf("Removing old app bundle backup: %s", oldBundlePath)

			if err := os.RemoveAll(oldBundlePath); err != nil {
				log.Printf("Failed to remove old app bundle backup: %v", err)
			}
		}
	}()

	if err := os.Rename(newBundlePath, appBundlePath); err != nil {
		rollback = true
		return fmt.Errorf("rename new app bundle: %w", err)
	}

	return nil
}

func (su *SelfUpdater) applyUpdateForWindowsOrLinux(tmpFile string) error {
	tempDir, err := os.MkdirTemp("", "unarchive-*")
	if err != nil {
		return fmt.Errorf("create temp unarchive dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := unarchive(tmpFile, tempDir); err != nil {
		return fmt.Errorf("unzip file: %w", err)
	}

	currentExecPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	oldExecPath := generateBackupName(currentExecPath)
	if err := os.Rename(currentExecPath, oldExecPath); err != nil {
		return fmt.Errorf("rename current executable to backup: %w", err)
	}

	rollback := false
	defer func() {
		if rollback {
			log.Printf("Restoring original executable from: %s", oldExecPath)
			if err := os.Rename(oldExecPath, currentExecPath); err != nil {
				log.Printf("Failed to restore original executable: %v", err)
			}
		} else {
			log.Printf("Removing backup executable: %s", oldExecPath)
			if err := os.Remove(oldExecPath); err != nil {
				log.Printf("Failed to remove backup executable: %v", err)

				log.Printf("Attempting to hide file: %s", oldExecPath)
				err = hideFile(oldExecPath)
				if err != nil {
					log.Printf("Failed to hide backup executable: %v", err)
				}
			}
		}
	}()

	if err := replaceExecutable(tempDir); err != nil {
		rollback = true
		return fmt.Errorf("replace executable: %w", err)
	}

	return nil
}

func (su *SelfUpdater) restartApplication(ctx context.Context) error {
	cmd := exec.Command(os.Args[0], os.Args[1:]...) // #nosec G204
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart application: %w", err)
	}
	wailsruntime.Quit(ctx)
	return nil
}

// hideFile moves the file at the given path to a temporary directory in case it cannot be removed.
func hideFile(path string) error {
	tmpDir := os.TempDir()
	newPath := filepath.Join(tmpDir, filepath.Base(path))

	if err := os.Rename(path, newPath); err != nil {
		return fmt.Errorf("move file to temp storage: %w", err)
	}

	log.Printf("Moved file to temporary storage: %s", newPath)
	return nil
}

func findAppBundleInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && filepath.Ext(entry.Name()) == ".app" {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no .app bundle found in directory: %s", dir)
}

func generateBackupName(originalName string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s.backup-%d", originalName, timestamp)
}
