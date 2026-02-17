package util

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "15 minutes",
			duration: 15 * time.Minute,
			expected: "15 minutes",
		},
		{
			name:     "1 minute",
			duration: 1 * time.Minute,
			expected: "1 minute",
		},
		{
			name:     "1 hour",
			duration: 1 * time.Hour,
			expected: "1 hour",
		},
		{
			name:     "2 hours",
			duration: 2 * time.Hour,
			expected: "2 hours",
		},
		{
			name:     "24 hours",
			duration: 24 * time.Hour,
			expected: "1 day",
		},
		{
			name:     "1 day",
			duration: 24 * time.Hour,
			expected: "1 day",
		},
		{
			name:     "2 days",
			duration: 2 * 24 * time.Hour,
			expected: "2 days",
		},
		{
			name:     "7 days",
			duration: 7 * 24 * time.Hour,
			expected: "7 days",
		},
		{
			name:     "59 minutes",
			duration: 59 * time.Minute,
			expected: "59 minutes",
		},
		{
			name:     "90 minutes",
			duration: 90 * time.Minute,
			expected: "1 hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}
