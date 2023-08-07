package config

import (
	"embed"
	"encoding/json"
	"os"
	"path"
	"runtime"
)

const APP_NAME = "zen"

var Config struct {
	Filter struct {
		FilterLists []string `json:"filterLists"`
	} `json:"matcher"`
	ConfigDir string
	DataDir   string
}

//go:embed default-config.json
var defaultConfig embed.FS

func init() {
	configDir, err := getConfigDir()
	if err != nil {
		panic(err)
	}
	dataDir, err := getDataDir()
	if err != nil {
		panic(err)
	}

	configFile := path.Join(configDir, "config.json")
	var configData []byte
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configData, err = defaultConfig.ReadFile("default-config.json")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(configFile, configData, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		// config file exists, read it
		configData, err = os.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
	}
	err = json.Unmarshal(configData, &Config)
	if err != nil {
		panic(err)
	}
	Config.ConfigDir = configDir
	Config.DataDir = dataDir
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var dir string
	if runtime.GOOS == "windows" {
		dir = path.Join(homeDir, "AppData", "Roaming", APP_NAME)
	} else {
		// unix, linux, macos
		dir = path.Join(homeDir, ".config", APP_NAME)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func getDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var dir string
	if runtime.GOOS == "windows" {
		dir = path.Join(homeDir, "AppData", "Local", APP_NAME)
	} else {
		// unix, linux, macos
		dir = path.Join(homeDir, ".local", "share", APP_NAME)
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}

	return dir, nil
}
