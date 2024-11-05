package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/natefinch/lumberjack.v2"
)

func (a *App) SetupLogger() error {
	logsDir, err := getLogsDir(a.name)
	if err != nil {
		return err
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "application.log"),
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     1,
		Compress:   true,
	})

	return nil
}

func (a *App) OpenLogsFolder() error {
	logsDir, err := getLogsDir(a.name)
	if err != nil {
		return fmt.Errorf("failed to get logs directory: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", logsDir).Start()
	case "darwin":
		return exec.Command("open", logsDir).Start()
	case "linux":
		return exec.Command("xdg-open", logsDir).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func getLogsDir(appName string) (string, error) {
	var path string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		path = filepath.Join(os.Getenv("LOCALAPPDATA"), appName, "Logs")
	case "darwin":
		path = filepath.Join(homeDir, "Library", "Logs", appName)
	case "linux":
		path = filepath.Join(homeDir, ".local", "share", appName, "logs")
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	return path, nil
}
