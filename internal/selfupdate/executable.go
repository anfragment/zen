package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func replaceWindowsExecutable(tempDir string) error {
	expectedExecName := appName
	if runtime.GOOS == "windows" {
		expectedExecName += ".exe"
	}
	newExecPath := filepath.Join(tempDir, expectedExecName)

	if _, err := os.Stat(newExecPath); os.IsNotExist(err) {
		return fmt.Errorf("expected executable '%s' not found", expectedExecName)
	}

	currentExecPath, err := getExecPath()
	if err != nil {
		return fmt.Errorf("get exec path: %w", err)
	}

	if err := os.Rename(newExecPath, currentExecPath); err != nil {
		return fmt.Errorf("move new executable: %w", err)
	}

	return nil
}

func getExecPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}

	// https://github.com/golang/go/issues/40966
	if runtime.GOOS != "windows" {
		if execPath, err = filepath.EvalSymlinks(execPath); err != nil {
			return "", fmt.Errorf("eval symlinks: %w", err)
		}
	}

	return execPath, nil
}

func findAppBundlePath(execPath string) string {
	dir := filepath.Dir(execPath)
	for dir != "/" {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

// https://github.com/inconshreveable/go-update/blob/master/hide_windows.go#L8
func hideWindowsFile(path string) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttributes := kernel32.NewProc("SetFileAttributesW")

	utf16Path, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("convert path to UTF-16: %w", err)
	}

	ret, _, err := setFileAttributes.Call(uintptr(unsafe.Pointer(utf16Path)), uintptr(0x2))
	if ret == 0 {
		return fmt.Errorf("set file attributes: %w", err)
	}

	return nil
}
