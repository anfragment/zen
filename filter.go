package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type Filter struct {
	// hosts is a map of hostnames we want to filter.
	hosts   map[string]struct{}
	hostsMu sync.RWMutex
}

func NewFilter() *Filter {
	return &Filter{
		hosts: make(map[string]struct{}),
	}
}

var (
	// domainCG is a capture group for a domain name.
	domainCG            = `((?:[\da-z][\da-z_-]*\.)+[\da-z-]*[a-z])`
	reHosts             = regexp.MustCompile(fmt.Sprintf(`^(?:0\.0\.0\.0|127\.0\.0\.1) %s`, domainCG))
	reHostsDomainIgnore = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
	reFilterDomain      = regexp.MustCompile(fmt.Sprintf(`^\|\|%s\^$`, domainCG))
)

// AddRemoteFilters parses the hosts files at the given URLs and adds the hosts
// listed in them to the filter.
func (f *Filter) AddRemoteFilters(urls []string) error {
	var wg sync.WaitGroup

	errCount := 0
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := f.AddRemoteFilter(url); err != nil {
				errCount++
				log.Printf("error adding remote hosts: %v", err)
			}
		}(url)
	}

	wg.Wait()
	if errCount == len(urls) {
		return fmt.Errorf("error adding all remote hosts")
	}
	return nil
}

// AddRemoteFilter parses the hosts file at the given URL and adds the hosts
// listed in it to the filter.
func (f *Filter) AddRemoteFilter(url string) error {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	c := 0
	for scanner.Scan() {
		if m := reHosts.FindStringSubmatch(scanner.Text()); m != nil {
			host := m[1]
			if reHostsDomainIgnore.MatchString(host) {
				continue
			}
			f.addHost(host)
			c++
			continue
		}
		if m := reFilterDomain.FindStringSubmatch(scanner.Text()); m != nil {
			host := m[1]
			f.addHost(host)
			c++
			continue
		}
	}

	log.Printf("Added %d hosts from %s in %s", c, url, time.Since(start))
	return nil
}

func (f *Filter) addHost(host string) {
	f.hostsMu.Lock()
	defer f.hostsMu.Unlock()
	f.hosts[host] = struct{}{}
}

// IsBlocked returns true if the given host is blocked by the filter.
func (f *Filter) IsBlocked(host string) bool {
	f.hostsMu.RLock()
	defer f.hostsMu.RUnlock()
	_, ok := f.hosts[host]
	return ok
}
