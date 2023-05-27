package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"
)

type CertManager struct {
	caCert *x509.Certificate
	caKey  any
}

func NewCertManager(caCertFile, caKeyFile string) (*CertManager, error) {
	mgr := &CertManager{}
	if err := mgr.loadX509KeyPair(caCertFile, caKeyFile); err != nil {
		return nil, fmt.Errorf("failed to load CA certificate/key: %w", err)
	}
	return mgr, nil
}

// loadX509KeyPair loads a certificate/key pair from the given files.n
func (mgr *CertManager) loadX509KeyPair(certFile, keyFile string) (err error) {
	cf, err := ioutil.ReadFile(certFile)
	if err != nil {
		return err
	}

	kf, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return err
	}
	certBlock, _ := pem.Decode(cf)
	mgr.caCert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(kf)
	mgr.caKey, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	return nil
}

// CreateCert creates a TLS certificate for the given DNS names, signed by the
// CA certificate/key pair.
func (mgr *CertManager) CreateCert(dnsNames []string, hoursValid int) (*tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Zen Proxy"},
		},
		DNSNames:  dnsNames,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(hoursValid) * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, mgr.caCert, &privateKey.PublicKey, mgr.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		return nil, fmt.Errorf("failed to encode certificate to PEM")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %w", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemCert == nil {
		return nil, fmt.Errorf("failed to encode private key to PEM")
	}
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate: %w", err)
	}
	return &tlsCert, nil
}
