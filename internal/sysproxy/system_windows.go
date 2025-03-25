package sysproxy

import (
	_ "embed"
	"fmt"
	"log"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	wininet                       = windows.NewLazySystemDLL("wininet.dll")
	internetSetOption             = wininet.NewProc("InternetSetOptionW")
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37

	//go:embed exclusions/windows.txt
	platformSpecificExcludedHosts []byte
)

func setSystemProxy(pacURL string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("set ProxyEnable: %v", err)
	}
	if err := k.SetStringValue("AutoConfigURL", pacURL); err != nil {
		return fmt.Errorf("set AutoConfigURL: %v", err)
	}

	callInternetSetOption(internetOptionSettingsChanged)
	callInternetSetOption(internetOptionRefresh)

	return nil
}

func unsetSystemProxy() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.DeleteValue("AutoConfigURL"); err != nil {
		return fmt.Errorf("delete AutoConfigURL: %v", err)
	}

	callInternetSetOption(internetOptionSettingsChanged)
	callInternetSetOption(internetOptionRefresh)

	return nil
}

func callInternetSetOption(dwOption int) {
	ret, _, err := internetSetOption.Call(0, uintptr(dwOption), 0, 0)
	if ret == 0 {
		log.Printf("failed to call InternetSetOption with option %d: %v", dwOption, err)
	}
}
