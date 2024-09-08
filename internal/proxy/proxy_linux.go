package proxy

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var exclusionListURLs = []string{
	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/common.txt",
}

var (
	ErrUnsupportedDesktopEnvironment = errors.New("system proxy configuration is currently only supported on GNOME")
)

func (p *Proxy) setSystemProxy() error {
	if binaryExists("gsettings") {
		commands := [][]string{
			[]string{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
			[]string{"gsettings", "set", "org.gnome.system.proxy.http", "host", "127.0.0.1"},
			[]string{"gsettings", "set", "org.gnome.system.proxy.http", "port", fmt.Sprint(p.port)},
			[]string{"gsettings", "set", "org.gnome.system.proxy.https", "host", "127.0.0.1"},
			[]string{"gsettings", "set", "org.gnome.system.proxy.https", "port", fmt.Sprint(p.port)},
		}

		for _, command := range commands {
			cmd := exec.Command(command[0], command[1:]...) // #nosec G204
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("run system proxy command (%s): %w\n%s", strings.Join(command, " "), err, out)
			}
		}
		return nil
	}
	// TODO: add support for other desktop environments

	return ErrUnsupportedDesktopEnvironment
}

func (p *Proxy) unsetSystemProxy() error {
	cmd := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unset system proxy: %v\n%s", err, out)
	}

	return nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
