package sysproxy

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// exclusionListURLs = []string{
	// 	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/common.txt",
	// 	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/apple.txt",
	// }
	reInterfaceName = regexp.MustCompile(`^[\w\d]+$`)
	networkService  string
)

// setSystemProxy sets the system proxy to the proxy address.
func setSystemProxy(pacURL string) error {
	cmd := exec.Command("sh", "-c", "scutil --nwi | grep 'Network interfaces' | cut -d ' ' -f 3")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("get default interface: %v\n%s", err, out)
	}
	interfaceName := strings.TrimSpace(string(out))
	if len(interfaceName) == 0 {
		return errors.New("no default interface found")
	}
	if !reInterfaceName.MatchString(interfaceName) {
		// I am pretty sure that interface names can only contain alphanumeric characters,
		// but just to be sure not to introduce a shell injection vulnerability, let's check it.
		return fmt.Errorf("invalid interface name: %s", interfaceName)
	}

	cmd = exec.Command("sh", "-c", fmt.Sprintf("networksetup -listnetworkserviceorder | grep %s -B 1 | head -n 1 | cut -d ' ' -f 2-", interfaceName)) // #nosec G204 -- Interface name is validated above
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("get network service: %v\n%s", err, out)
	}
	networkService = strings.TrimSpace(string(out))
	if len(networkService) == 0 {
		return errors.New("no network service found")
	}

	cmd = exec.Command("networksetup", "-setautoproxyurl", networkService, pacURL)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("set web proxy for network service %q: %v\n%s", networkService, err, out)
	}

	cmd = exec.Command("networksetup", "-setautoproxystate", networkService, "on")
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("enable web proxy for network service %q: %v\n%s", networkService, err, out)
	}

	return nil
}

func unsetSystemProxy() error {
	if networkService == "" {
		return fmt.Errorf("trying to unset system proxy without setting it first")
	}

	cmd := exec.Command("networksetup", "-setautoproxyurl", networkService, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unset web proxy (network service: %s): %v\n%s", networkService, err, out)
	}

	networkService = ""

	return nil
}
