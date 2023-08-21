package proxy

import (
	"fmt"
	"os/exec"
	"strings"
)

var interfaceName string

// setSystemProxy sets the system proxy to the proxy address
func (p *Proxy) setSystemProxy() error {
	cmd := exec.Command("sh", "-c", "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get default interface: %v\n%s", err, out)
	}

	interfaceName = strings.TrimSpace(string(out))
	cmd = exec.Command("networksetup", "-setwebproxy", interfaceName, p.host, fmt.Sprint(p.port))
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set web proxy: %v\n%s", err, out)
	}

	cmd = exec.Command("networksetup", "-setsecurewebproxy", interfaceName, p.host, fmt.Sprint(p.port))
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set secure web proxy: %v\n%s", err, out)
	}

	return nil
}

func (p *Proxy) unsetSystemProxy() error {
	if interfaceName == "" {
		return fmt.Errorf("trying to unset system proxy without setting it first")
	}

	cmd := exec.Command("networksetup", "-setwebproxystate", interfaceName, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unset web proxy: %v\n%s", err, out)
	}

	cmd = exec.Command("networksetup", "-setsecurewebproxystate", interfaceName, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unset secure web proxy: %v\n%s", err, out)
	}

	return nil
}
