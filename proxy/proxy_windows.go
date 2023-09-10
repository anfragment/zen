package proxy

import (
	"fmt"
	"log"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var exclusionListURLs = []string{
	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/common.txt",
}

var (
	wininet                          = windows.NewLazySystemDLL("wininet.dll")
	internetSetOption                = wininet.NewProc("InternetSetOptionW")
	INTERNET_OPTION_SETTINGS_CHANGED = 39
	INTERNET_OPTION_REFRESH          = 37
)

func (p *Proxy) setSystemProxy() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}

	if err := k.SetStringValue("ProxyServer", fmt.Sprintf("%s:%d", p.host, p.port)); err != nil {
		if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
			log.Printf("failed to disable proxy during error handling: %v", err)
		}
		return err
	}

	callInternetSetOption(INTERNET_OPTION_SETTINGS_CHANGED)
	callInternetSetOption(INTERNET_OPTION_REFRESH)

	return nil
}

func (p *Proxy) unsetSystemProxy() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return err
	}

	callInternetSetOption(INTERNET_OPTION_SETTINGS_CHANGED)
	callInternetSetOption(INTERNET_OPTION_REFRESH)

	return nil
}

func callInternetSetOption(dwOption int) {
	ret, _, err := internetSetOption.Call(0, uintptr(dwOption), 0, 0)
	if ret == 0 {
		log.Printf("failed to call InternetSetOption with option %d: %v", dwOption, err)
	}
}
