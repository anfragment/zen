package filterliststore

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/anfragment/zen/internal/filter/filterliststore/diskcache"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	// expiresRegex matches lines like "! Expires: 4 days", supporting formats such as: "4 days", "12 hours", "5d", and "18h".
	expiresRegex = regexp.MustCompile(`(?i)! Expires:\s*(\d+)\s*(days?|hours?|d|h)?`)

	// ignoreLineRegex matches comments and [Adblock Plus 2.0]-style headers.
	ignoreLineRegex = regexp.MustCompile(`^(?:!|\[|#([^#%]|$))`)
)

type FilterListStore struct {
	cache *diskcache.Cache
}

func NewFilterListStore() *FilterListStore {
	cache, err := diskcache.New()
	if err != nil {
		log.Fatalf("failed to init diskcache: %v", err)
	}

	return &FilterListStore{
		cache: cache,
	}
}

func (st *FilterListStore) Get(url string) ([]byte, error) {
	if content, ok := st.cache.Load(url); ok {
		return content, nil
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response: %s", resp.Status)
	}

	var comments [][]byte
	var rules [][]byte

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := slices.Clone(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		if ignoreLineRegex.Match(line) {
			comments = append(comments, line)
		} else {
			rules = append(rules, line)
		}
	}

	rulesBytes := bytes.Join(rules, []byte("\n"))
	expiry := extractExpiryTimestamp(bytes.Join(comments, []byte("\n")), time.Now())

	if err := st.cache.Save(url, expiry, rulesBytes); err != nil {
		return nil, fmt.Errorf("cache save: %w", err)
	}

	return rulesBytes, nil
}

func extractExpiryTimestamp(content []byte, now time.Time) time.Time {
	defaultExpiry := now.Add(24 * time.Hour)
	lines := bytes.Split(content, []byte("\n"))

	for _, line := range lines {
		matches := expiresRegex.FindSubmatch(line)
		if len(matches) >= 2 {
			amount, err := strconv.Atoi(string(matches[1]))
			if err != nil {
				continue
			}
			if amount == 0 {
				continue
			}

			unit := "days"
			if len(matches) >= 3 {
				unit = strings.ToLower(strings.TrimSpace(string(matches[2])))
			}

			switch unit {
			case "day", "days", "d":
				return now.Add(time.Duration(amount) * 24 * time.Hour)
			case "hour", "hours", "h":
				return now.Add(time.Duration(amount) * time.Hour)
			}
		}
	}

	return defaultExpiry
}
