package hostmatch

import (
	"errors"
	"strings"
)

var (
	errNoEmptyPattern = errors.New("empty patterns are not allowed")
)

type hostnameStore[T any] interface {
	Add(hostnamePattern string, data T)
	Get(hostname string) []T
}

type HostMatcher[T comparable] struct {
	primaryStore      hostnameStore[T]
	generic           []T
	exceptionStore    hostnameStore[T]
	genericExceptions []T
}

func NewHostMatcher[T comparable]() *HostMatcher[T] {
	return &HostMatcher[T]{
		primaryStore:   newTrieStore[T](),
		exceptionStore: newTrieStore[T](),
	}
}

func (hm *HostMatcher[T]) AddPrimaryRule(hostnamePatterns string, data T) error {
	if len(hostnamePatterns) == 0 {
		hm.generic = append(hm.generic, data)
		return nil
	}

	patterns := strings.Split(hostnamePatterns, ",")
	for _, pattern := range patterns {
		if len(pattern) == 0 {
			return errNoEmptyPattern
		}
	}
	for _, pattern := range patterns {
		if pattern[0] == '~' {
			pattern = pattern[1:]
			hm.exceptionStore.Add(pattern, data)
			continue
		}

		hm.primaryStore.Add(pattern, data)
		if !strings.HasPrefix(pattern, "*.") {
			hm.primaryStore.Add("*."+pattern, data)
		}
	}

	return nil
}

func (hm *HostMatcher[T]) AddExceptionRule(hostnamePatterns string, data T) error {
	if len(hostnamePatterns) == 0 {
		hm.generic = append(hm.generic, data)
		return nil
	}

	patterns := strings.Split(hostnamePatterns, ",")
	for _, pattern := range patterns {
		if len(pattern) == 0 {
			return errNoEmptyPattern
		}

		hm.exceptionStore.Add(pattern, data)
		if !strings.HasPrefix(pattern, "*.") {
			hm.exceptionStore.Add("*."+pattern, data)
		}
	}

	return nil
}

func (hm *HostMatcher[T]) Get(hostname string) []T {
	primaryResults := hm.primaryStore.Get(hostname)
	generic := make([]T, len(hm.generic)+len(primaryResults))
	copy(generic, hm.generic)
	copy(generic[len(hm.generic):], primaryResults)

	exceptionResults := hm.exceptionStore.Get(hostname)
	exceptions := make([]T, len(hm.genericExceptions)+len(exceptionResults))
	copy(exceptions, hm.genericExceptions)
	copy(exceptions[len(hm.genericExceptions):], exceptionResults)

	exceptionRuleMap := make(map[T]struct{}, len(exceptions))
	for _, exception := range exceptions {
		exceptionRuleMap[exception] = struct{}{}
	}

	filtered := make([]T, 0, len(generic))
	for _, rule := range generic {
		if _, ok := exceptionRuleMap[rule]; !ok {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}
