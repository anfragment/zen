package cfg

import (
	"os"
	"path"
)

const (
	appFolderName = "Zen"
	configDirName = "Config"
)

func getConfigDir() (string, error) {
	// use LOCALAPPDATA instead of APPDATA because the config includes some machine-specific data
	// e.g. whether the CA has been installed
	if os.Getenv("LOCALAPPDATA") != "" {
		return path.Join(os.Getenv("LOCALAPPDATA"), appFolderName, configDirName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, "AppData", "Local", appFolderName, configDirName), nil
}

func getDataDir() (string, error) {
	if os.Getenv("LOCALAPPDATA") != "" {
		return path.Join(os.Getenv("LOCALAPPDATA"), appFolderName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homeDir, "AppData", "Local", appFolderName), nil
}
