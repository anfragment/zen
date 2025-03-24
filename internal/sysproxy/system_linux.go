package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

var exclusionListURLs = []string{
	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/common.txt",
}

func setSystemProxy(pacURL string) error {
	if !binaryExists("gsettings") {
		// TODO: add support for other desktop environments
		return ErrUnsupportedDesktopEnvironment
	}

	commands := [][]string{
		[]string{"gsettings", "set", "org.gnome.system.proxy", "autoconfig-url", pacURL},
		[]string{"gsettings", "set", "org.gnome.system.proxy", "mode", "auto"},
	}

	for _, command := range commands {
		cmd := exec.Command(command[0], command[1:]...) // #nosec G204
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run system proxy command %q: %v (%q)", strings.Join(command, " "), err, out)
		}
	}
	return nil
}

func unsetSystemProxy() error {
	if !binaryExists("gsettings") {
		// TODO: add support for other desktop environments
		return ErrUnsupportedDesktopEnvironment
	}

	cmd := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unset system proxy: %v (%q)", err, out)
	}

	return nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
