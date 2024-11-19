package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func replaceExecutable(tempDir string) error {
	expectedExecName := appName
	if runtime.GOOS == "windows" {
		expectedExecName += ".exe"
	}
	newExecPath := filepath.Join(tempDir, expectedExecName)

	if _, err := os.Stat(newExecPath); os.IsNotExist(err) {
		return fmt.Errorf("expected executable '%s' not found", expectedExecName)
	}

	currentExecPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	if err := os.Rename(newExecPath, currentExecPath); err != nil {
		return fmt.Errorf("move new executable: %w", err)
	}

	return nil
}

func getExecPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}

	// https://github.com/golang/go/issues/40966
	if runtime.GOOS != "windows" {
		if execPath, err = filepath.EvalSymlinks(execPath); err != nil {
			return "", fmt.Errorf("eval symlinks: %w", err)
		}
	}

	return execPath, nil
}

func findAppBundlePath(execPath string) string {
	dir := filepath.Dir(execPath)
	for dir != "/" {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
