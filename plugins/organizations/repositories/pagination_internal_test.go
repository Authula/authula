package repositories

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
)

func TestPageLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{name: "positive limit is preserved", limit: 25, expected: 25},
		{name: "zero limit falls back to the default", limit: 0, expected: pagination.DefaultLimit},
		{name: "negative limit falls back to the default", limit: -1, expected: pagination.DefaultLimit},
		{name: "limit at the maximum is preserved", limit: pagination.MaxLimit, expected: pagination.MaxLimit},
		{name: "limit above the maximum is capped", limit: 100000, expected: pagination.MaxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, pageLimit(tt.limit))
		})
	}
}

func TestPageOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		page     int
		limit    int
		expected int
	}{
		{name: "first page starts at zero", page: 1, limit: 10, expected: 0},
		{name: "third page skips two pages", page: 3, limit: 10, expected: 20},
		{name: "page zero never produces a negative offset", page: 0, limit: 10, expected: 0},
		{name: "negative page never produces a negative offset", page: -5, limit: 10, expected: 0},
		{name: "non positive limit produces no offset", page: 3, limit: 0, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, pageOffset(tt.page, tt.limit))
		})
	}
}
