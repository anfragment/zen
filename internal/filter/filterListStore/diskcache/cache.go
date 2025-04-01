package diskcache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user cache dir: %w", err)
	}
	return filepath.Join(base, "zen", "filters"), nil
}

func hashURL(url string) string {
	sum := md5.Sum([]byte(url))
	return hex.EncodeToString(sum[:])
}

func getLatestFile(hash string) (string, int64) {
	dir, err := cacheDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "diskcache: %v\n", err)
		return nil, false
	}
	files, _ := os.ReadDir(dir)
	prefix := hash + "-"
	var latest string
	var latestTs int64
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), prefix) {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(f.Name(), ".cache"), "-")
		if len(parts) != 2 {
			continue
		}
		ts, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		if ts > latestTs {
			latestTs = ts
			latest = filepath.Join(dir, f.Name())
		}
	}
	return latest, latestTs
}

func Load(url string) ([]byte, bool) {
	hash := hashURL(url)
	path, ts := getLatestFile(hash)
	if path == "" || time.Since(time.Unix(ts, 0)) > cacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func Save(url string, content []byte) error {
	hash := hashURL(url)
	ts := time.Now().Unix()
	name := fmt.Sprintf("%s-%d.cache", hash, ts)
	fullPath := filepath.Join(cacheDir(), name)

	if err := os.MkdirAll(cacheDir(), 0755); err != nil {
		return fmt.Errorf("make cache dir: %w", err)
	}
	return os.WriteFile(fullPath, content, 0644)
}
