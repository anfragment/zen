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

type hostMatcher[T comparable] struct {
	primaryStore      hostnameStore[T]
	generic           []T
	exceptionStore    hostnameStore[T]
	genericExceptions []T
}

func NewHostMatcher[T comparable]() *hostMatcher[T] {
	return &hostMatcher[T]{
		primaryStore:   newTrieStore[T](),
		exceptionStore: newTrieStore[T](),
	}
}

func (hm *hostMatcher[T]) AddPrimaryRule(hostnamePatterns string, data T) error {
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
		} else {
			hm.primaryStore.Add(pattern, data)
		}
	}

	return nil
}

func (hm *hostMatcher[T]) AddExceptionRule(hostnamePatterns string, data T) error {
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
	}

	return nil
}

func (hm *hostMatcher[T]) Get(hostname string) []T {
	generic := append(hm.generic, hm.primaryStore.Get(hostname)...)
	exceptions := append(hm.genericExceptions, hm.exceptionStore.Get(hostname)...)

	exceptionRuleMap := make(map[T]any, len(exceptions))
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
