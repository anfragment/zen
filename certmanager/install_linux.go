package certmanager

import (
	"fmt"
	"os"
	"os/exec"
)

func (cm *CertManager) install() error {
	var systemTrustCommand []string
	var systemTrustPath string
	if pathExists("/etc/pki/ca-trust/source/anchors/") {
		systemTrustPath = "/etc/pki/ca-trust/source/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-trust", "extract"}
	} else if pathExists("/usr/local/share/ca-certificates/") {
		systemTrustPath = "/usr/local/share/ca-certificates/%s.crt"
		systemTrustCommand = []string{"update-ca-certificates"}
	} else if pathExists("/etc/ca-certificates/trust-source/anchors/") {
		systemTrustPath = "/etc/ca-certificates/trust-source/anchors/%s.crt"
		systemTrustCommand = []string{"trust", "extract-compat"}
	} else if pathExists("/usr/share/pki/trust/anchors") {
		systemTrustPath = "/usr/share/pki/trust/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-certificates"}
	}

	filename := fmt.Sprintf(systemTrustPath, "zen-rootCA")
	err := os.WriteFile(filename, cm.CertData, 0644)
	if err != nil {
		return fmt.Errorf("failed to install CA: %v", err)
	}

	cmd := exec.Command("sudo", systemTrustCommand...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install CA: %v\n%s", err, out)
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
