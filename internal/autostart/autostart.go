package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager manages automatic startup of the app on user login.
type Manager struct{}

// getExecPath returns the path to the currently running executable.
func getExecPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	if execPath, err = filepath.EvalSymlinks(execPath); err != nil {
		return "", fmt.Errorf("eval symlinks: %w", err)
	}
	return execPath, nil
}
