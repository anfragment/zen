package cfg

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
)

var (
	// ConfigDir is the path to the directory storing the application configuration.
	ConfigDir string
	// DataDir is the path to the directory storing the application data.
	DataDir string
)

//go:embed default-config.json
var defaultConfig embed.FS

// Config stores and manages the configuration for the application.
// Although all fields are public, this is only for use by the JSON marshaller.
// All access to the Config should be done through the exported methods.
type Config struct {
	sync.RWMutex

	Filter struct {
		FilterLists []FilterList `json:"filterLists"`
	} `json:"filter"`
	Certmanager struct {
		CAInstalled bool `json:"caInstalled"`
	} `json:"certmanager"`
	Proxy struct {
		Port int `json:"port"`
	} `json:"proxy"`

	// firstLaunch is true if the application is being run for the first time.
	firstLaunch bool
}

type FilterList struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func init() {
	var err error
	ConfigDir, err = getConfigDir()
	if err != nil {
		log.Fatalf("failed to get config dir: %v", err)
	}
	stat, err := os.Stat(ConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(ConfigDir, 0755); err != nil {
				log.Fatalf("failed to create config dir: %v", err)
			}
		} else {
			log.Fatalf("failed to stat config dir: %v", err)
		}
	}
	if !stat.IsDir() {
		log.Fatalf("config dir is not a directory")
	}

	DataDir, err = getDataDir()
	if err != nil {
		log.Fatalf("failed to get data dir: %v", err)
	}
	stat, err = os.Stat(DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(DataDir, 0755); err != nil {
				log.Fatalf("failed to create data dir: %v", err)
			}
		} else {
			log.Fatalf("failed to stat data dir: %v", err)
		}
	}
	if !stat.IsDir() {
		log.Fatalf("data dir is not a directory")
	}
}

func NewConfig() (*Config, error) {
	c := &Config{}

	configFile := path.Join(ConfigDir, "config.json")
	var configData []byte
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		configData, err = os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	} else {
		configData, err = defaultConfig.ReadFile("default-config.json")
		if err != nil {
			return nil, fmt.Errorf("failed to read default config file: %v", err)
		}
		if err := os.WriteFile(configFile, configData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write config file: %v", err)
		}
		c.firstLaunch = true
	}

	if err := json.Unmarshal(configData, c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return c, nil
}

// Save saves the config to disk.
// It is not thread-safe, and should only be called if the caller has
// a lock on the config.
func (c *Config) Save() error {
	configData, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configFile := path.Join(ConfigDir, "config.json")
	err = os.WriteFile(configFile, configData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// GetFilterLists returns the list of enabled filter lists.
func (c *Config) GetFilterLists() []FilterList {
	c.RLock()
	defer c.RUnlock()

	return c.Filter.FilterLists
}

// AddFilterList adds a new filter list to the list of enabled filter lists.
func (c *Config) AddFilterList(list FilterList) string {
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
func (c *Config) RemoveFilterList(url string) string {
	c.Lock()
	defer c.Unlock()

	for i, filterList := range c.Filter.FilterLists {
		if filterList.URL == url {
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
func (c *Config) ToggleFilterList(url string, enabled bool) string {
	c.Lock()
	defer c.Unlock()

	for i, filterList := range c.Filter.FilterLists {
		if filterList.URL == url {
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
func (c *Config) GetPort() int {
	c.RLock()
	defer c.RUnlock()

	return c.Proxy.Port
}

// SetPort sets the port the proxy is set to listen on.
func (c *Config) SetPort(port int) string {
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
func (c *Config) GetCAInstalled() bool {
	c.RLock()
	defer c.RUnlock()

	return c.Certmanager.CAInstalled
}

// SetCAInstalled sets whether the CA is installed.
func (c *Config) SetCAInstalled(caInstalled bool) {
	c.Lock()
	defer c.Unlock()

	c.Certmanager.CAInstalled = caInstalled
	if err := c.Save(); err != nil {
		log.Printf("failed to save config: %v", err)
	}
}
