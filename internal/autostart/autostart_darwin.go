package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	reverseDNSAppName = "net.zenprivacy.zen"
	plistTemplate     = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>Label</key>
		<string>{{.ReverseDNSAppName}}</string>
		<key>Program</key>
		<string>{{.Program}}</string>
		<key>ProgramArguments</key>
			<array>
				<string>--start</string>
			</array>
		<key>RunAtLoad</key>
		<true/>
		<key>AbandonProcessGroup</key>
		<true/>
		<key>ProcessType</key>
		<string>Interactive</string>
	</dict>
</plist>`
)

type plistTemplateParameters struct {
	Program           string
	ReverseDNSAppName string
}

func (m Manager) IsEnabled() (bool, error) {
	plistPath, err := m.getPath()
	if err != nil {
		return false, fmt.Errorf("get launch plist path: %w", err)
	}

	_, err = os.Stat(plistPath)
	return err == nil, nil
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

	launchDir, err := getLaunchDir()
	if err != nil {
		return fmt.Errorf("get launch dir: %w", err)
	}
	plistPath, err := m.getPath()
	if err != nil {
		return fmt.Errorf("get launch plist path: %w", err)
	}

	if err := os.MkdirAll(launchDir, 0755); err != nil {
		return fmt.Errorf("create launch dir: %w", err)
	}
	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("create plist file: %w", err)
	}
	defer f.Close()

	t := template.Must(template.New("plist").Parse(plistTemplate))

	if err := t.Execute(f, plistTemplateParameters{
		Program: execPath,
	}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

func (m Manager) Disable() error {
	if enabled, err := m.IsEnabled(); err != nil {
		return fmt.Errorf("check enabled: %w", err)
	} else if !enabled {
		return nil
	}

	plistPath, err := m.getPath()
	if err != nil {
		return fmt.Errorf("get launch plist path: %w", err)
	}
	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("remove plist: %w", err)
	}

	return nil
}

func (m Manager) getPath() (string, error) {
	launchDir, err := getLaunchDir()
	if err != nil {
		return "", fmt.Errorf("get launch dir: %w", err)
	}

	return filepath.Join(launchDir, reverseDNSAppName+".plist"), nil
}

func getLaunchDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	return filepath.Join(homeDir, "Library", "LaunchAgents"), nil
}
