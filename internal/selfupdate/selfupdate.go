package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"syscall"
	"time"

	"github.com/inconshreveable/go-update"
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

func unarchiveTar(gz io.Reader, dest string) error {
	tarReader := tar.NewReader(gz)

	dest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		targetPath := filepath.Join(dest, header.Name)
		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("error resolving file path: %w", err)
		}

		// Check if the target path is within the destination directory
		if !strings.HasPrefix(absTargetPath, dest) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to write file data: %w", err)
			}

			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set file permissions: %w", err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		default:
			return fmt.Errorf("unsupported type: %v in tar archive", header.Typeflag)
		}
	}

	return nil
}

func unarchiveTarGz(src io.Reader, dest string) error {
	gzReader, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return unarchiveTar(gzReader, dest)
}

func extractAndWriteFile(f *zip.File, dest, rootFolder string) error {
	relativePath := strings.TrimPrefix(f.Name, rootFolder+"/")
	if relativePath == "" {
		return nil
	}

	path := filepath.Join(dest, relativePath)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return os.Chmod(path, f.Mode())
}

func uncompressTo(src io.Reader, url, dest string) error {
	if strings.HasSuffix(url, ".zip") {
		buf, err := io.ReadAll(src)
		if err != nil {
			return fmt.Errorf("failed to create buffer for zip file: %s", err)
		}

		r := bytes.NewReader(buf)
		z, err := zip.NewReader(r, r.Size())
		if err != nil {
			return fmt.Errorf("failed to uncompress zip file: %s", err)
		}

		// wont work in windows/linux, todo fix
		// Extract all files and directories in the zip archive
		for _, file := range z.File {
			err = extractAndWriteFile(file, dest, "Zen.app")
			if err != nil {
				return err
			}
		}

		return nil
	} else if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		log.Println("Uncompressing tar.gz file", url)

		gz, err := gzip.NewReader(src)
		if err != nil {
			return fmt.Errorf("failed to uncompress .tar.gz file: %s", err)
		}

		return unarchiveTarGz(gz, dest)
	}

	log.Println("Uncompression is not needed", url)
	return nil
}

func (su *SelfUpdater) downloadFromURL(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Add("Accept", "application/octet-stream")

	res, err := su.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return res.Body, nil
}

func (su *SelfUpdater) ApplyUpdate() error {
	rel, err := su.checkForUpdates()
	if err != nil {
		return err
	}

	isNewer, err := su.isNewer(rel.Version)
	if err != nil {
		return err
	}

	if !isNewer {
		return nil
	}

	updateFile, err := su.downloadFromURL(rel.AssetURL)
	if err != nil {
		return err
	}
	defer updateFile.Close()

	hashMatches, err := checkSHA256(rel.SHA256, updateFile)
	if err != nil {
		return err
	}
	if !hashMatches {
		return errors.New("SHA256 checksum mismatch")
	}

	applyUpdate(runtime.GOOS, updateFile, rel.AssetURL)
	log.Println("Update applied")

	return nil
}

func applyUpdate(goos string, updateFile io.Reader, url string) error {
	if goos == "darwin" {
		appPath := "/Applications/Zen.app"

		// remove for now, implement backup later
		err := os.Remove(appPath)
		if err != nil {
			log.Println("Failed to rename app backup", err)
		}

		err = uncompressTo(updateFile, url, appPath)
		if err != nil {
			return err
		}

		err = restartApp()
		if err != nil {
			return fmt.Errorf("failed to restart app: %w", err)
		}

	} else if goos == "windows" || goos == "linux" {
		err := update.Apply(updateFile, update.Options{})

		if err != nil {
			return fmt.Errorf("failed to apply update: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported OS: %s", goos)
	}

	return nil
}

func restartApp() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command(execPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start new process: %w", err)
		}

		// Give time for the new process to start, then exit the current one
		time.Sleep(2 * time.Second)
		os.Exit(0)

	} else {
		// for unix-like systems
		return syscall.Exec(execPath, os.Args, os.Environ())
	}

	return nil
}

func checkSHA256(expectedHash string, reader io.Reader) (bool, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return false, err
	}

	calculatedHash := hasher.Sum(nil)
	calculatedHashString := hex.EncodeToString(calculatedHash)

	return calculatedHashString == expectedHash, nil
}
