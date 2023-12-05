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
	"fmt"
	"os"
	"os/exec"
)

var (
	systemTrustFilename string
	systemTrustCommand  []string
)

func init() {
	if pathExists("/etc/pki/ca-trust/source/anchors/") {
		systemTrustFilename = "/etc/pki/ca-trust/source/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-trust", "extract"}
	} else if pathExists("/usr/local/share/ca-certificates/") {
		systemTrustFilename = "/usr/local/share/ca-certificates/%s.crt"
		systemTrustCommand = []string{"update-ca-certificates"}
	} else if pathExists("/etc/ca-certificates/trust-source/anchors/") {
		systemTrustFilename = "/etc/ca-certificates/trust-source/anchors/%s.crt"
		systemTrustCommand = []string{"trust", "extract-compat"}
	} else if pathExists("/usr/share/pki/trust/anchors") {
		systemTrustFilename = "/usr/share/pki/trust/anchors/%s.pem"
		systemTrustCommand = []string{"update-ca-certificates"}
	}
}

// installCA installs the root CA into the system trust store.
func (cm *CertManager) installCA() error {
	if systemTrustFilename == "" {
		return fmt.Errorf("installing to the system store is not yet supported on this Linux")
	}

	filename := fmt.Sprintf(systemTrustFilename, "zen-rootCA")
	cmd := exec.Command("pkexec", "tee", filename)
	cmd.Stdin = bytes.NewReader(cm.certData)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("write cert to %s: %v\n%s", filename, err, out)
	}

	cmd = exec.Command("pkexec", systemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("system trust command: %v\n%s", err, out)
	}

	return nil
}

func (cm *CertManager) uninstallCA() error {
	if systemTrustFilename == "" {
		return nil
	}

	filename := fmt.Sprintf(systemTrustFilename, "zen-rootCA")
	cmd := exec.Command("pkexec", "rm", "-f", filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove cert from %s: %v\n%s", filename, err, out)
	}

	cmd = exec.Command("pkexec", systemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("system trust command: %v\n%s", err, out)
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
