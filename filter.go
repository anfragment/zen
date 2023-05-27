package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

type Filter struct {
	// hosts is a map of hostnames we want to filter.
	hosts map[string]struct{}
}

func NewFilter() *Filter {
	return &Filter{
		hosts: make(map[string]struct{}),
	}
}

var sinkholeIPs = []string{
	"0.0.0.0",
	"::",
	"127.0.0.1",
	"::1",
}

// AddRemoteHosts parses the hosts file at the given URL and adds the hosts
// listed in it to the filter.
func (f *Filter) AddRemoteHosts(url string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch hosts file: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 || text[0] == '#' {
			continue
		}
		for _, ip := range sinkholeIPs {
			if strings.HasPrefix(text, ip) {
				f.hosts[strings.TrimSpace(text[len(ip):])] = struct{}{}
			}
		}
	}
	return nil
}

// IsBlocked returns true if the given host is blocked by the filter.
func (f *Filter) IsBlocked(host string) bool {
	_, ok := f.hosts[host]
	return ok
}
