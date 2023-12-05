/*
This file contains code from the mkcert project,
licensed under the BSD 3-Clause License:
Copyright (c) 2018 The mkcert Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:
- Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
- Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package certmanager

import (
	"bytes"
	"encoding/asn1"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/anfragment/zen/config"
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

// installCA installs the root CA certificate into the system trust store.
func (cm *CertManager) installCA() error {
	cmd := elevate.WithPrompt("Authorize Zen to install the root CA certificate").Command(
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

// uninstallCA removes the root CA certificate from the system trust store.
func (cm *CertManager) uninstallCA() error {
	cmd := elevate.WithPrompt("Authorize Zen to remove the root CA certificate").Command(
		"security", "delete-certificate", "-c", cm.cert.Subject.CommonName, "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("failed to delete root CA certificate: %v\n%s", err, out)
		return fmt.Errorf("delete root CA certificate: %v\n%s", err, out)
	}

	folderName := path.Join(config.Config.DataDir, certsFolderName())
	if err := os.RemoveAll(folderName); err != nil {
		log.Printf("failed to remove certs folder: %v", err)
		return fmt.Errorf("remove certs folder: %v", err)
	}

	return nil
}
