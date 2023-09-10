package proxy

import (
	"fmt"
	"os/exec"
	"strings"
)

var exclusionListURLs = []string{
	"https://raw.githubusercontent.com/anfragment/zen/main/proxy/exclusions/common.txt",
}

func (p *Proxy) setSystemProxy() error {
	if binaryExists("gsettings") {
		commands := [][]string{
			[]string{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
			[]string{"gsettings", "set", "org.gnome.system.proxy.http", "host", p.host},
			[]string{"gsettings", "set", "org.gnome.system.proxy.http", "port", fmt.Sprint(p.port)},
			[]string{"gsettings", "set", "org.gnome.system.proxy.https", "host", p.host},
			[]string{"gsettings", "set", "org.gnome.system.proxy.https", "port", fmt.Sprint(p.port)},
		}

		for _, command := range commands {
			cmd := exec.Command(command[0], command[1:]...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %v\n%s", strings.Join(command, " "), err, out)
			}
		}
		return nil
	}
	// Proxy configuration on non-GNOME systems is finicky
	// and I don't have the expertise to figure it out.
	// TODO: add support for other environments

	return fmt.Errorf("system proxy configuration is currently only supported on gnome")
}

func (p *Proxy) unsetSystemProxy() error {
	command := []string{"gsettings", "set", "org.gnome.system.proxy", "mode", "none"}
	cmd := exec.Command(command[0], command[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v\n%s", strings.Join(command, " "), err, out)
	}

	return nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
