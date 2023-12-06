package config

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"path"
	"sync"
)

// Config is the singleton config instance.
var Config config

type filterList struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Url     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// config stores and manages the configuration for the application.
// Although all fields are public, this is only for use by the JSON marshaller.
// All access to the config should be done through the exported methods.
//
// Methods that get called by the frontend should be annotated with @frontend.
type config struct {
	sync.RWMutex
	Filter struct {
		FilterLists []filterList `json:"filterLists"`
	} `json:"filter"`
	Certmanager struct {
		CAInstalled bool `json:"caInstalled"`
	} `json:"certmanager"`
	Proxy struct {
		Port uint16 `json:"port"`
	} `json:"proxy"`
	ConfigDir string `json:"-"`
	DataDir   string `json:"-"`
}

// Save saves the config to disk.
// It is not thread-safe, and should only be called if the caller has
// a lock on the config.
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

// GetFilterLists returns the list of enabled filter lists.
//
// @frontend
func (c *config) GetFilterLists() []filterList {
	c.RLock()
	defer c.RUnlock()

	return c.Filter.FilterLists
}

// AddFilterList adds a new filter list to the list of enabled filter lists.
//
// @frontend
func (c *config) AddFilterList(list filterList) string {
	c.Lock()
	defer c.Unlock()

	c.Filter.FilterLists = append(c.Filter.FilterLists, list)
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
		return err.Error()
	}
	return ""
}

// RemoveFilterList removes a filter list from the list of enabled filter lists.
//
// @frontend
func (c *config) RemoveFilterList(url string) string {
	c.Lock()
	defer c.Unlock()

	for i, filterList := range c.Filter.FilterLists {
		if filterList.Url == url {
			c.Filter.FilterLists = append(c.Filter.FilterLists[:i], c.Filter.FilterLists[i+1:]...)
			break
		}
	}
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
		return err.Error()
	}
	return ""
}

// ToggleFilterList toggles the enabled state of a filter list.
//
// @frontend
func (c *config) ToggleFilterList(url string, enabled bool) string {
	c.Lock()
	defer c.Unlock()

	for i, filterList := range c.Filter.FilterLists {
		if filterList.Url == url {
			c.Filter.FilterLists[i].Enabled = enabled
			break
		}
	}
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
		return err.Error()
	}
	return ""
}

// GetPort returns the port the proxy is set to listen on.
//
// @frontend
func (c *config) GetPort() uint16 {
	c.RLock()
	defer c.RUnlock()

	return c.Proxy.Port
}

// SetPort sets the port the proxy is set to listen on.
//
// @frontend
func (c *config) SetPort(port uint16) string {
	c.Lock()
	defer c.Unlock()

	c.Proxy.Port = port
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
		return err.Error()
	}
	return ""
}

// GetCAInstalled returns whether the CA is installed.
func (c *config) GetCAInstalled() bool {
	c.RLock()
	defer c.RUnlock()

	return c.Certmanager.CAInstalled
}

// SetCAInstalled sets whether the CA is installed.
func (c *config) SetCAInstalled(caInstalled bool) {
	c.Lock()
	defer c.Unlock()

	c.Certmanager.CAInstalled = caInstalled
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
	}
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
