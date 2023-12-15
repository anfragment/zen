package certmanager

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestPutAndGet tests that certificates can be stored in the cache and retrieved.
func TestPutAndGet(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(100, time.Hour)
	cert := &tls.Certificate{}
	cache.Put("example.com", time.Now().Add(24*time.Hour), cert)
	if cache.Get("example.com") != cert {
		t.Errorf("Expected the retrieved certificate to be the same as the one put in")
	}
}

// TestPutMultipleTimes tests that certificates can be stored in the cache multiple times.
func TestPutMultipleTimes(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(100, time.Hour)
	cert1 := &tls.Certificate{}
	cert2 := &tls.Certificate{}
	cache.Put("example.com", time.Now().Add(24*time.Hour), cert1)
	cache.Put("example.com", time.Now().Add(24*time.Hour), cert2)
	if cache.Get("example.com") != cert2 {
		t.Errorf("Expected the retrieved certificate to be the same as the one put in")
	}
}

// TestMultipleCerts tests that multiple certificates can be stored in the cache.
func TestMultipleCerts(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(1000, time.Hour)
	certs := make([]*tls.Certificate, 1000)

	expiresAt := time.Now().Add(24 * time.Hour)
	for i := 0; i < 1000; i++ {
		certs[i] = &tls.Certificate{}
		cache.Put(fmt.Sprintf("example%d.com", i), expiresAt, certs[i])
	}

	for _, i := range rand.Perm(1000) {
		if cache.Get(fmt.Sprintf("example%d.com", i)) != certs[i] {
			t.Fatalf("Expected the retrieved certificate to be the same as the one put in. Failure at index %d.", i)
		}
	}
}

// TestExpiration tests that certificates expire after the given TTL.
// May introduce flakiness if the test machine is under heavy load.
func TestExpiration(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(4000, time.Second)

	checkTTL := func(ttl time.Duration, doneC chan<- struct{}, errC chan<- error) {
		now := time.Now()
		certs := make([]*tls.Certificate, 1000)
		for i := 0; i < len(certs); i++ {
			certs[i] = &tls.Certificate{}
			cache.Put(fmt.Sprintf("%d.%d.example.com", i, ttl), now.Add(ttl), certs[i])
		}

		<-time.After(ttl / 2)
		for _, i := range rand.Perm(1000) {
			if cache.Get(fmt.Sprintf("%d.%d.example.com", i, ttl)) != certs[i] {
				errC <- fmt.Errorf("Expected the retrieved certificate to be the same as the one put in. Failure at index %d.", i)
				return
			}
		}

		<-time.After((ttl / 2) + time.Second)
		for _, i := range rand.Perm(1000) {
			if cache.Get(fmt.Sprintf("%d.%d.example.com", i, ttl)) != nil {
				errC <- fmt.Errorf("Expected the certificate to expire. Failure at index %d.", i)
				return
			}
		}

		doneC <- struct{}{}
	}

	doneC := make(chan struct{}, 3)
	errC := make(chan error, 1)

	go checkTTL(3*time.Second, doneC, errC)
	go checkTTL(5*time.Second, doneC, errC)
	go checkTTL(10*time.Second, doneC, errC)

	for i := 0; i < 3; i++ {
		select {
		case <-doneC:
		case err := <-errC:
			t.Fatal(err)
		case <-time.After(15 * time.Second):
			t.Fatal("Timed out")
		}
	}
}

// TestGetExpired tests that expired certificates are not returned.
func TestGetExpired(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(1000, time.Hour)
	cert := &tls.Certificate{}
	cache.Put("example.com", time.Now().Add(-time.Hour), cert)
	if cache.Get("example.com") != nil {
		t.Errorf("Expected the retrieved certificate to be nil")
	}
}

// TestLRU tests that the LRU eviction policy works.
func TestLRU(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(2000, time.Hour)

	certsToBeEvicted := make([]*tls.Certificate, 1000)
	certsToBeKept := make([]*tls.Certificate, 1000)
	expiresAt := time.Now().Add(24 * time.Hour)

	for i := 0; i < 1000; i++ {
		certsToBeEvicted[i] = &tls.Certificate{}
		cache.Put(fmt.Sprintf("%d.evict.com", i), expiresAt, certsToBeEvicted[i])
	}

	for i := 0; i < 1000; i++ {
		certsToBeKept[i] = &tls.Certificate{}
		cache.Put(fmt.Sprintf("%d.keep.com", i), expiresAt, certsToBeKept[i])
	}

	for _, i := range rand.Perm(1000) {
		cache.Get(fmt.Sprintf("%d.keep.com", i))
	}

	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("%d.new.com", i), expiresAt, &tls.Certificate{})
	}

	for _, i := range rand.Perm(1000) {
		if cache.Get(fmt.Sprintf("%d.evict.com", i)) != nil {
			t.Fatalf("Expected the certificate to be evicted. Failure at index %d.", i)
		}
	}

	for _, i := range rand.Perm(1000) {
		if cache.Get(fmt.Sprintf("%d.keep.com", i)) != certsToBeKept[i] {
			t.Fatalf("Expected the certificate to be kept. Failure at index %d.", i)
		}
	}
}

// TestPurge tests that the cache can be purged.
func TestPurge(t *testing.T) {
	t.Parallel()

	cache := NewCertLRUCache(1000, time.Hour)

	cert := &tls.Certificate{}
	cache.Put("example.com", time.Now().Add(24*time.Hour), cert)
	if cache.Get("example.com") != cert {
		t.Errorf("Expected the retrieved certificate to be the same as the one put in")
	}

	cache.Purge()
	if cache.Get("example.com") != nil {
		t.Errorf("Expected the retrieved certificate to be nil after purge")
	}
}
