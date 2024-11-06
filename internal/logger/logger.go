package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	appName = "Zen"
)

func SetupLogger() error {
	logsDir, err := getLogsDir(appName)
	if err != nil {
		return fmt.Errorf("failed to get logs directory: %w", err)
	}

	fileLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "application.log"),
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     1,
		Compress:   true,
	}

	log.SetOutput(io.MultiWriter(os.Stdout, fileLogger))

	return nil
}

func OpenLogsDirectory() error {
	logsDir, err := getLogsDir(appName)
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
		panic("unsupported platform")
	}
}

func getLogsDir(appName string) (string, error) {
	var path string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		path = filepath.Join(os.Getenv("LOCALAPPDATA"), appName, "Logs")
	case "darwin":
		path = filepath.Join(homeDir, "Library", "Logs", appName)
	case "linux":
		path = filepath.Join(homeDir, ".local", "share", appName, "logs")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	return path, nil
}
