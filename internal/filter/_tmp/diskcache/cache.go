package diskcache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

func getCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "zen", "filters"), nil
}

func hashURL(url string) string {
	sum := md5.Sum([]byte(url))
	return hex.EncodeToString(sum[:])
}

func getLatestCacheFile(hash string) (string, int64, error) {
	dir, err := getCacheDir()
	if err != nil {
		return "", 0, fmt.Errorf("get cache dir: %w", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, fmt.Errorf("read dir: %w", err)
	}

	var latestFile string
	var latestTimestamp int64
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), hash) {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(f.Name(), ".cache"), "-")
		if len(parts) != 2 {
			continue
		}
		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		if timestamp > latestTimestamp {
			latestTimestamp = timestamp
			latestFile = filepath.Join(dir, f.Name())
		}
	}
	return latestFile, latestTimestamp, nil
}

func Load(url string) ([]byte, bool) {
	hash := hashURL(url)
	path, timestamp, err := getLatestCacheFile(hash)
	if err != nil {
		log.Printf("error getting cache file: %v", err)
		return nil, false
	}
	if path == "" || timestamp == 0 {
		return nil, false
	}

	if time.Since(time.Unix(timestamp, 0)) > cacheTTL {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("error reading cache file: %v", err)
		return nil, false
	}
	return data, true
}

func Save(url string, content []byte) error {
	hash := hashURL(url)
	timestamp := time.Now().Unix()
	name := fmt.Sprintf("%s-%d.cache", hash, timestamp)
	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("get cache dir: %w", err)
	}

	fullPath := filepath.Join(cacheDir, name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("make cache dir: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
