package certmanager

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
	"math/big"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/anfragment/zen/config"
	"github.com/hectane/go-acl"
)

// CertManager manages the root CA certificate and key for the proxy.
type CertManager struct {
	CertData []byte
	KeyData  []byte
	certPath string
	cert     *x509.Certificate
	keyPath  string
	key      crypto.PrivateKey
}

const (
	caName  = "rootCA.pem"
	keyName = "rootCA-key.pem"
)

// NewCertManager creates and initializes a new CertManager.
func NewCertManager() (*CertManager, error) {
	folderName := path.Join(config.Config.DataDir, certsFolderName())

	cm := &CertManager{
		certPath: path.Join(folderName, caName),
		keyPath:  path.Join(folderName, keyName),
	}

	if config.Config.Certmanager.CAInstalled {
		if err := cm.loadCA(); err != nil {
			return nil, fmt.Errorf("failed to load CA: %v", err)
		}
	} else {
		if err := os.Remove(cm.certPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove old CA certificate: %v", err)
		}
		if err := os.Remove(cm.keyPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove old CA key: %v", err)
		}
		if err := os.MkdirAll(folderName, 0755); err != nil {
			return nil, fmt.Errorf("failed to create certs folder: %v", err)
		}
		if err := cm.newCA(); err != nil {
			return nil, fmt.Errorf("failed to generate CA: %v", err)
		}
		if err := cm.loadCA(); err != nil {
			return nil, fmt.Errorf("failed to load CA: %v", err)
		}
		if err := cm.install(); err != nil {
			return nil, fmt.Errorf("failed to install CA into system trust store: %v", err)
		}
		config.Config.Certmanager.CAInstalled = true
		config.Config.Save()
	}

	return cm, nil
}

// loadCA loads the existing CA certificate and key into memory.
func (cm *CertManager) loadCA() error {
	if _, err := os.Stat(cm.certPath); os.IsNotExist(err) {
		return fmt.Errorf("CA certificate not found at %s", cm.certPath)
	}
	if _, err := os.Stat(cm.keyPath); os.IsNotExist(err) {
		return fmt.Errorf("CA key not found at %s", cm.keyPath)
	}

	var err error
	cm.CertData, err = os.ReadFile(cm.certPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}
	certDERBlock, _ := pem.Decode(cm.CertData)
	if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("failed to decode CA certificate: unexpected content")
	}
	cm.cert, err = x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	cm.KeyData, err = os.ReadFile(cm.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}
	keyDERBlock, _ := pem.Decode(cm.KeyData)
	if keyDERBlock == nil || keyDERBlock.Type != "PRIVATE KEY" {
		return fmt.Errorf("failed to decode CA certificate: unexpected content")
	}
	cm.key, err = x509.ParsePKCS8PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	return nil
}

// newCA creates a new CA certificate/key pair and saves it to disk.
func (cm *CertManager) newCA() error {
	priv, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}
	pub := priv.Public()

	spkiASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %v", err)
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %v", err)
	}

	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %v", err)
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
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %v", err)
	}
	err = os.WriteFile(cm.keyPath, pem.EncodeToMemory(
		&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0400)
	if err != nil {
		return fmt.Errorf("failed to save private key: %v", err)
	}
	if runtime.GOOS == "windows" {
		if err := acl.Chmod(cm.keyPath, 0400); err != nil {
			return fmt.Errorf("failed to set permissions on private key: %v", err)
		}
	}

	err = os.WriteFile(cm.certPath, pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0644)
	if err != nil {
		return fmt.Errorf("failed to save certificate: %v", err)
	}
	if runtime.GOOS == "windows" {
		if err := acl.Chmod(cm.certPath, 0644); err != nil {
			return fmt.Errorf("failed to set permissions on certificate: %v", err)
		}
	}

	return nil
}

func certsFolderName() string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// following a general convention of using capitalized folder names on Windows and macOS
		return "Certs"
	}
	return "certs"
}
