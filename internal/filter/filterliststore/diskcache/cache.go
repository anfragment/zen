package diskcache

import (
	"crypto/md5" // #nosec G501 -- MD5 is used to hash data, not for cryptographic purposes.
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	expiresAt time.Time
	filename  string
}

type urlHash string

type Cache struct {
	dir       string
	entriesMu sync.RWMutex
	entries   map[urlHash]cacheEntry
}

func New() (*Cache, error) {
	dir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		dir:     dir,
		entries: make(map[urlHash]cacheEntry),
	}

	if err := cache.loadFromDisk(); err != nil {
		log.Printf("error loading cache from disk: %v", err)
	}

	return cache, nil
}

func (c *Cache) loadFromDisk() error {
	dirEntries, err := os.ReadDir(c.dir)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(c.dir, 0755); err != nil {
			return fmt.Errorf("create cache dir: %v", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("read cache dir: %v", err)
	}

	for _, e := range dirEntries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if !strings.HasSuffix(name, ".cache.txt") {
			continue
		}

		parts := strings.Split(strings.TrimSuffix(e.Name(), ".cache.txt"), "-")
		if len(parts) != 2 {
			continue
		}

		hash := urlHash(parts[0])
		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		if time.Unix(timestamp, 0).Before(time.Now()) {
			// TODO: delete the file
			continue
		}

		c.entries[hash] = cacheEntry{
			expiresAt: time.Unix(timestamp, 0),
			filename:  name,
		}
	}

	return nil
}

func (c *Cache) Save(url string, content []byte, expiresAt time.Time) error {
	c.entriesMu.Lock()
	defer c.entriesMu.Unlock()

	hash := hashURL(url)
	filename := fmt.Sprintf("%s-%d.cache.txt", hash, expiresAt.Unix())

	fullPath := filepath.Join(c.dir, filename)
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return err
	}

	c.entries[hash] = cacheEntry{
		expiresAt: expiresAt,
		filename:  filename,
	}

	return nil
}

func (c *Cache) Load(url string) (io.ReadCloser, error) {
	c.entriesMu.RLock()
	defer c.entriesMu.RUnlock()

	hash := hashURL(url)
	entry, ok := c.entries[hash]
	if !ok {
		return nil, nil
	}

	if entry.expiresAt.Before(time.Now()) {
		return nil, nil
	}

	f, err := os.Open(filepath.Join(c.dir, entry.filename))
	if err != nil {
		return nil, fmt.Errorf("open cache file: %w", err)
	}

	return f, nil
}

func getCacheDir() (string, error) {
	var appName string
	switch runtime.GOOS {
	case "darwin", "windows":
		appName = "Zen"
	case "linux":
		appName = "zen"
	default:
		panic("unsupported platform")
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, "filters"), nil
}

func hashURL(url string) urlHash {
	sum := md5.Sum([]byte(url)) // #nosec G401
	return urlHash(hex.EncodeToString(sum[:]))
}
