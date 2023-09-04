package config

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"path"
)

const APP_NAME = "zen"

var Config config

type config struct {
	Filter struct {
		FilterLists []string `json:"filterLists"`
	} `json:"filter"`
	Certmanager struct {
		CAInstalled bool `json:"caInstalled"`
	} `json:"certmanager"`
	ConfigDir string `json:"-"`
	DataDir   string `json:"-"`
}

func (c *config) Save() error {
	configData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	configFile := path.Join(c.ConfigDir, "config.json")
	err = os.WriteFile(configFile, configData, 0644)
	if err != nil {
		return err
	}
	return nil
}

//go:embed default-config.json
var defaultConfig embed.FS

func init() {
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("failed to get config dir: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("failed to create config dir: %v", err)
	}

	dataDir, err := getDataDir()
	if err != nil {
		log.Fatalf("failed to get data dir: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	configFile := path.Join(configDir, "config.json")
	var configData []byte
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		configData, err = os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("failed to read config file: %v", err)
		}
	} else {
		configData, err = defaultConfig.ReadFile("default-config.json")
		if err != nil {
			log.Fatalf("failed to read default config file: %v", err)
		}
		if err := os.WriteFile(configFile, configData, 0644); err != nil {
			log.Fatalf("failed to write config file: %v", err)
		}
	}

	if err := json.Unmarshal(configData, &Config); err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	Config.ConfigDir = configDir
	Config.DataDir = dataDir
}
