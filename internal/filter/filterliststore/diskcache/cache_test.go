package diskcache

import (
	"testing"
	"time"
)

func TestExtractExpiryTimestamp(t *testing.T) {
	now := time.Date(2025, 4, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "4 days",
			input:    "! Expires: 4 days",
			expected: now.Add(4 * 24 * time.Hour),
		},
		{
			name:     "12 hours",
			input:    "! Expires: 12 hours",
			expected: now.Add(12 * time.Hour),
		},
		{
			name:     "5d shorthand",
			input:    "! Expires: 5d",
			expected: now.Add(5 * 24 * time.Hour),
		},
		{
			name:     "18h shorthand",
			input:    "! Expires: 18h",
			expected: now.Add(18 * time.Hour),
		},
		{
			name:     "invalid input, fallback to default",
			input:    "no expires here",
			expected: now.Add(24 * time.Hour),
		},
		{
			name:     "zero duration, fallback to default",
			input:    "! Expires: 0 days",
			expected: now.Add(24 * time.Hour),
		},
		{
			name:     "unsupported unit, fallback to default",
			input:    "! Expires: 10 weeks",
			expected: now.Add(24 * time.Hour),
		},
		{
			name:     "whitespace and mixed case",
			input:    "   ! expires: 2 HOURS   ",
			expected: now.Add(2 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExpiryTimestamp([]byte(tt.input), now)
			if !got.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
