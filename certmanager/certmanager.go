package certmanager

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

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/anfragment/zen/config"
	"github.com/hectane/go-acl"
)

const (
	caName  = "rootCA.pem"
	keyName = "rootCA-key.pem"
	// cacheMaxSize is the maximum number of certificates the cache will store.
	//
	// Considering that a single tls.Certificate is about 1.7KB, this means that the cache
	// can store 5800 certificates in about 10MB of memory.
	cacheMaxSize = 5800
	// cacheCleanupInterval is the interval at which the cache is cleaned up.
	cacheCleanupInterval = 5 * time.Minute
)

// CertManager manages the root CA certificate and key for the proxy.
type CertManager struct {
	certData  []byte
	keyData   []byte
	certPath  string
	cert      *x509.Certificate
	keyPath   string
	key       crypto.PrivateKey
	certCache *certLRUCache
}

func NewCertManager() (*CertManager, error) {
	cm := &CertManager{}

	folderName := path.Join(config.Config.DataDir, certsFolderName())
	cm.certPath = path.Join(folderName, caName)
	cm.keyPath = path.Join(folderName, keyName)
	cm.certCache = newCertLRUCache(cacheMaxSize, cacheCleanupInterval)

	if config.Config.GetCAInstalled() {
		if err := cm.loadCA(); err != nil {
			return nil, fmt.Errorf("CA load: %v", err)
		}
		return cm, nil
	}

	if err := os.Remove(cm.certPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove existing CA cert: %v", err)
	}
	if err := os.Remove(cm.keyPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove existing CA key: %v", err)
	}
	if err := os.MkdirAll(folderName, 0755); err != nil {
		return nil, fmt.Errorf("create certs folder: %v", err)
	}
	if err := cm.newCA(); err != nil {
		return nil, fmt.Errorf("create new CA: %v", err)
	}
	if err := cm.loadCA(); err != nil {
		return nil, fmt.Errorf("CA load: %v", err)
	}
	if err := cm.installCA(); err != nil {
		return nil, fmt.Errorf("install CA: %v", err)
	}
	config.Config.SetCAInstalled(true)

	return cm, nil
}

// loadCA loads the existing CA certificate and key into memory.
func (cm *CertManager) loadCA() error {
	if _, err := os.Stat(cm.certPath); os.IsNotExist(err) {
		return fmt.Errorf("CA cert does not exist at %s", cm.certPath)
	}
	if _, err := os.Stat(cm.keyPath); os.IsNotExist(err) {
		return fmt.Errorf("CA key does not exist at %s", cm.keyPath)
	}

	var err error
	cm.certData, err = os.ReadFile(cm.certPath)
	if err != nil {
		return fmt.Errorf("read CA cert: %v", err)
	}
	certDERBlock, _ := pem.Decode(cm.certData)
	if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("invalid CA cert: type mismatch")
	}
	cm.cert, err = x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA cert: %v", err)
	}

	cm.keyData, err = os.ReadFile(cm.keyPath)
	if err != nil {
		return fmt.Errorf("read CA key: %v", err)
	}
	keyDERBlock, _ := pem.Decode(cm.keyData)
	if keyDERBlock == nil || keyDERBlock.Type != "PRIVATE KEY" {
		return fmt.Errorf("invalid CA key: type mismatch")
	}
	cm.key, err = x509.ParsePKCS8PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA key: %v", err)
	}

	return nil
}

// newCA creates a new CA certificate/key pair and saves it to disk.
func (cm *CertManager) newCA() error {
	priv, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return fmt.Errorf("generate key: %v", err)
	}
	pub := priv.Public()

	spkiASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return fmt.Errorf("marshal public key: %v", err)
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return fmt.Errorf("unmarshal public key: %v", err)
	}

	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("generate serial number: %v", err)
	}

	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Zen Proxy"},
			OrganizationalUnit: []string{"Zen Proxy CA"},
			CommonName:         "Zen Proxy CA",
		},
		SubjectKeyId: skid[:],

		NotAfter:  time.Now().AddDate(32, 0, 0),
		NotBefore: time.Now(),

		KeyUsage: x509.KeyUsageCertSign,

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, tpl, tpl, pub, priv)
	if err != nil {
		return fmt.Errorf("create certificate: %v", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal private key: %v", err)
	}
	err = os.WriteFile(cm.keyPath, pem.EncodeToMemory(
		&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0600)
	if err != nil {
		return fmt.Errorf("write private key at %s: %v", cm.keyPath, err)
	}
	if runtime.GOOS == "windows" {
		// 0500 to allow the user to delete the file
		if err := acl.Chmod(cm.keyPath, 0600); err != nil {
			return fmt.Errorf("chmod private key at %s: %v", cm.keyPath, err)
		}
	}

	err = os.WriteFile(cm.certPath, pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0644)
	if err != nil {
		return fmt.Errorf("write certificate at %s: %v", cm.certPath, err)
	}
	if runtime.GOOS == "windows" {
		if err := acl.Chmod(cm.certPath, 0644); err != nil {
			return fmt.Errorf("chmod certificate at %s: %v", cm.certPath, err)
		}
	}

	return nil
}

// ClearCache removes all cached certificates.
func (cm *CertManager) PurgeCache() {
	cm.certCache.Purge()
}

// UninstallCA wraps platform-specific uninstallCA methods.
func (cm *CertManager) UninstallCA() string {
	if !config.Config.GetCAInstalled() {
		return "CA is not installed"
	}

	if err := cm.uninstallCA(); err != nil {
		log.Printf("failed to uninstall CA: %v", err)
		return err.Error()
	}

	config.Config.SetCAInstalled(false)

	cm.certData = nil
	cm.keyData = nil
	cm.certPath = ""
	cm.cert = nil
	cm.keyPath = ""
	cm.key = nil
	cm.certCache = nil
	cm.PurgeCache()

	return ""
}

func certsFolderName() string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// following a general convention of using capitalized folder names on Windows and macOS
		return "Certs"
	}
	return "certs"
}
