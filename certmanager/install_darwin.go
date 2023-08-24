package certmanager

import (
	"bytes"
	"encoding/asn1"
	"fmt"
	"os"
	"os/exec"

	"github.com/getlantern/elevate"
	"howett.net/plist"
)

// https://github.com/golang/go/issues/24652#issuecomment-399826583
var trustSettings []interface{}
var _, _ = plist.Unmarshal(trustSettingsData, &trustSettings)
var trustSettingsData = []byte(`
<array>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAED
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>sslServer</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAEC
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>basicX509</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
</array>
`)

func (cm *CertManager) install() error {
	cmd := elevate.WithPrompt("Authorize Zen to install the root CA certificate")
		"security", "add-trusted-cert", "-d", "-k", "/Library/Keychains/System.keychain", cm.certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("add-trusted-cert: %v\n%s", err, out)
	}

	// Make trustSettings explicit, as older Go does not know the defaults.
	// https://github.com/golang/go/issues/24652

	plistFile, err := os.CreateTemp("", "trust-settings")
	if err != nil {
		return fmt.Errorf("create temporary plist file: %v", err)
	}
	defer os.Remove(plistFile.Name())

	cmd = exec.Command("security", "trust-settings-export", "-d", plistFile.Name())
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trust-settings-export: %v\n%s", err, out)
	}

	plistData, err := os.ReadFile(plistFile.Name())
	if err != nil {
		return fmt.Errorf("read plist file: %v", err)
	}
	var plistRoot map[string]interface{}
	_, err = plist.Unmarshal(plistData, &plistRoot)
	if err != nil {
		return fmt.Errorf("parse plist file: %v", err)
	}

	rootSubjectASN1, _ := asn1.Marshal(cm.cert.Subject.ToRDNSequence())

	if plistRoot["trustVersion"].(uint64) != 1 {
		return fmt.Errorf("unexpected trustVersion: %v", plistRoot["trustVersion"])
	}
	trustList := plistRoot["trustList"].(map[string]interface{})
	for key := range trustList {
		entry := trustList[key].(map[string]interface{})
		if _, ok := entry["issuerName"]; !ok {
			continue
		}
		issuerName := entry["issuerName"].([]byte)
		if !bytes.Equal(rootSubjectASN1, issuerName) {
			continue
		}
		entry["trustSettings"] = trustSettings
		break
	}

	plistData, err = plist.MarshalIndent(plistRoot, plist.XMLFormat, "\t")
	if err != nil {
		return fmt.Errorf("create plist data: %v", err)
	}
	err = os.WriteFile(plistFile.Name(), plistData, 0600)
	if err != nil {
		return fmt.Errorf("write plist file: %v", err)
	}
	cmd = exec.Command("security", "trust-settings-import", "-d", plistFile.Name())
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trust-settings-import: %v\n%s", err, out)
	}

	return nil
}
