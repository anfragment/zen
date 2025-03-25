package sysproxy

import (
	_ "embed"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	reInterfaceName = regexp.MustCompile(`^[\w\d]+$`)
	networkService  string
	//go:embed exclusions/darwin.txt
	platformSpecificExcludedHosts []byte
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

	cmd = exec.Command("networksetup", "-setwebproxystate", networkService, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unset web proxy for network service %q: %v (%q)", networkService, err, out)
	}

	cmd = exec.Command("networksetup", "-setsecurewebproxystate", networkService, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unset secure web proxy for network service %q: %v (%q)", networkService, err, out)
	}

	cmd = exec.Command("networksetup", "-setautoproxyurl", networkService, pacURL)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("set autoproxyurl to %q for network service %q: %v (%q)", pacURL, networkService, err, out)
	}

	// There's no need to set autoproxystate to on, as setting the URL already does that.

	return nil
}

func unsetSystemProxy() error {
	if networkService == "" {
		return errors.New("trying to unset system proxy without setting it first")
	}

	cmd := exec.Command("networksetup", "-setautoproxystate", networkService, "off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("set autoproxystate to off for network service %q: %v (%q)", networkService, err, out)
	}

	networkService = ""

	return nil
}
