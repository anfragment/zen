// autostart_windows.go provides autostart capabilities for Windows.
// To add the app to autostart, it creates a registry key under HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run.
//
// References:
// - https://learn.microsoft.com/en-us/windows/win32/setupapi/run-and-runonce-registry-keys

package autostart

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	regKey  = "Zen"
	regPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
)

func (m Manager) IsEnabled() (enabled bool, err error) {
	defer func() {
		if err != nil {
			log.Printf("error checking registry key: %s", err)
		}
	}()

	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	execPath, err := getExecPath()
	if err != nil {
		return false, fmt.Errorf("get exec path: %w", err)
	}

	value, _, err := key.GetStringValue(regKey)
	switch {
	case errors.Is(err, registry.ErrNotExist) || errors.Is(err, registry.ErrUnexpectedType):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("get string value: %w", err)
	}

	return strings.HasPrefix(value, execPath), nil // Use strings.HasPrefix to account for --start and any other future potential cli flags.
}

func (m Manager) Enable() (err error) {
	defer func() {
		if err != nil {
			log.Printf("error enabling autostart: %s", err)
		}
	}()

	if enabled, err := m.IsEnabled(); err != nil {
		return fmt.Errorf("check enabled: %w", err)
	} else if enabled {
		return nil
	}

	execPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	cmd := execPath + " " + "--start" + " " + "--hidden"

	if err := key.SetStringValue(regKey, cmd); err != nil {
		return fmt.Errorf("set string value: %w", err)
	}

	return nil
}

func (m Manager) Disable() (err error) {
	defer func() {
		if err != nil {
			log.Printf("error disabling autostart: %s", err)
		}
	}()

	if enabled, err := m.IsEnabled(); err != nil {
		return fmt.Errorf("check enabled: %w", err)
	} else if !enabled {
		return nil
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(regKey); err != nil {
		return fmt.Errorf("delete value: %w", err)
	}

	return nil
}
