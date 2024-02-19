package certgen

import (
	"crypto"
	"crypto/x509"
	"errors"
	"time"
)

const (
	// cacheMaxSize is the maximum number of certificates the cache will store.
	//
	// Considering that a single tls.Certificate is about 1.7KB, this means that the cache
	// can store 5800 certificates in about 10MB of memory.
	cacheMaxSize = 5800
	// cacheCleanupInterval is the interval at which the cache is cleaned up.
	cacheCleanupInterval = 5 * time.Minute
)

// certStore is an interface for getting a root CA certificate and its private key.
type certStore interface {
	GetCertificate() (*x509.Certificate, crypto.PrivateKey, error)
}

// CertGenerator allows for generating certificates for a given host.
type CertGenerator struct {
	cache *certLRUCache
	store certStore
}

func NewCertGenerator(certStore certStore) (*CertGenerator, error) {
	if certStore == nil {
		return nil, errors.New("certStore is nil")
	}

	return &CertGenerator{
		cache: newCertLRUCache(cacheMaxSize, cacheCleanupInterval),
		store: certStore,
	}, nil
}
