package cfg

import (
	"os"
	"path"
)

const (
	appFolderName = "zen"
)

// On Linux, we use the XDG Base Directory Specification:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html#variables

func getConfigDir() (string, error) {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return path.Join(os.Getenv("XDG_CONFIG_HOME"), appFolderName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, ".config", appFolderName), nil
}

func getDataDir() (string, error) {
	if os.Getenv("XDG_DATA_HOME") != "" {
		return path.Join(os.Getenv("XDG_DATA_HOME"), appFolderName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, ".local", "share", appFolderName), nil
}
