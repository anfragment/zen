package certmanager

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

const certTTL = 2 * time.Minute

// GetCertificate returns a self-signed certificate for the given host.
func (cm *CertManager) GetCertificate(host string) (*tls.Certificate, error) {
	// Using a map is probably a good enough idea due to the bursty nature of web activity.
	// Consider using a LRU cache if this turns out to be a bottleneck.
	cm.certCacheMu.RLock()
	cert, ok := cm.certCache[host]
	cm.certCacheMu.RUnlock()
	if ok {
		return &cert, nil
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %v", err)
	}

	expiry := time.Now().Add(certTTL)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Zen"},
		},
		DNSNames:  []string{host},
		NotBefore: time.Now(),
		NotAfter:  expiry,

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, cm.cert, &privateKey.PublicKey, cm.key)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %v", err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		return nil, fmt.Errorf("encode certificate to PEM")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemCert == nil {
		return nil, fmt.Errorf("encode key to PEM")
	}

	cert, err = tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %v", err)
	}

	cm.certCacheMu.Lock()
	cm.certCache[host] = cert
	cm.certCacheMu.Unlock()

	cm.ScheduleCacheCleanup(host, expiry)

	return &cert, nil
}

// ScheduleCacheCleanup clears the cache for the given host.
func (cm *CertManager) ScheduleCacheCleanup(host string, expiry time.Time) {
	go func() {
		time.Sleep(time.Until(expiry) - time.Second) // give it a second in case of a lock contention
		cm.certCacheMu.Lock()
		delete(cm.certCache, host)
		cm.certCacheMu.Unlock()
	}()
}
