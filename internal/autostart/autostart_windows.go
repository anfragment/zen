package autostart

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const registryKey = "Zen"

func (m Manager) IsEnabled() (bool, error) {
	key, err := openRegKey()
	if err != nil {
		return false, fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	execPath, err := getExecPath()
	if err != nil {
		return false, fmt.Errorf("get exec path: %w", err)
	}

	value, _, err := key.GetStringValue(registryKey)
	if err != nil {
		return false, fmt.Errorf("get string value: %w", err)
	}

	return value == execPath, nil
}

func (m Manager) Enable() error {
	if enabled, err := m.IsEnabled(); err != nil {
		return fmt.Errorf("check enabled: %w", err)
	} else if enabled {
		return nil
	}

	execPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	key, err := openRegKey()
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue(registryKey, execPath); err != nil {
		return fmt.Errorf("set string value: %w", err)
	}

	return nil
}

func (m Manager) Disable() error {
	if enabled, err := m.IsEnabled(); err != nil {
		return fmt.Errorf("check enabled: %w", err)
	} else if !enabled {
		return nil
	}

	key, err := openRegKey()
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(registryKey); err != nil {
		return fmt.Errorf("delete value: %w", err)
	}

	return nil
}

func openRegKey() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
}
