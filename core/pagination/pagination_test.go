package pagination_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
)

func TestClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   pagination.Params
		maxLimit int
		expected pagination.Params
	}{
		{
			name:     "valid params are untouched",
			params:   pagination.Params{Page: 1, Limit: 10},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: 10},
		},
		{
			name:     "page zero becomes the first page",
			params:   pagination.Params{Page: 0, Limit: 10},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: 10},
		},
		{
			name:     "negative page becomes the first page",
			params:   pagination.Params{Page: -5, Limit: 10},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: 10},
		},
		{
			name:     "zero limit falls back to the default limit",
			params:   pagination.Params{Page: 3, Limit: 0},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 3, Limit: pagination.DefaultLimit},
		},
		{
			name:     "negative limit falls back to the default limit",
			params:   pagination.Params{Page: 3, Limit: -1},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 3, Limit: pagination.DefaultLimit},
		},
		{
			name:     "limit exactly at the maximum is not capped",
			params:   pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
		},
		{
			name:     "limit above the maximum is capped",
			params:   pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit + 1},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
		},
		{
			name:     "an absurdly large limit is capped",
			params:   pagination.Params{Page: 1, Limit: 1_000_000},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
		},
		{
			name:     "a zero maximum falls back to the default maximum",
			params:   pagination.Params{Page: 1, Limit: 1_000_000},
			maxLimit: 0,
			expected: pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
		},
		{
			name:     "a negative maximum falls back to the default maximum",
			params:   pagination.Params{Page: 1, Limit: 1_000_000},
			maxLimit: -1,
			expected: pagination.Params{Page: 1, Limit: pagination.DefaultMaxLimit},
		},
		{
			name:     "a configured maximum above the default is honoured",
			params:   pagination.Params{Page: 1, Limit: 400},
			maxLimit: 500,
			expected: pagination.Params{Page: 1, Limit: 400},
		},
		{
			name:     "a limit above a configured maximum is capped to it",
			params:   pagination.Params{Page: 1, Limit: 501},
			maxLimit: 500,
			expected: pagination.Params{Page: 1, Limit: 500},
		},
		{
			name:     "a maximum below the default limit also caps the default",
			params:   pagination.Params{Page: 1, Limit: 0},
			maxLimit: 5,
			expected: pagination.Params{Page: 1, Limit: 5},
		},
		{
			name:     "high page numbers are legal",
			params:   pagination.Params{Page: 999999, Limit: 10},
			maxLimit: pagination.DefaultMaxLimit,
			expected: pagination.Params{Page: 999999, Limit: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, pagination.Clamp(tt.params, tt.maxLimit))
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		page               int
		limit              int
		total              int
		expectedTotalPages int
		expectedHasMore    bool
	}{
		{name: "empty result set", page: 1, limit: 10, total: 0, expectedTotalPages: 0, expectedHasMore: false},
		{name: "exactly one full page", page: 1, limit: 10, total: 10, expectedTotalPages: 1, expectedHasMore: false},
		{name: "one row spilling onto a second page", page: 1, limit: 10, total: 11, expectedTotalPages: 2, expectedHasMore: true},
		{name: "middle page has more", page: 2, limit: 25, total: 137, expectedTotalPages: 6, expectedHasMore: true},
		{name: "last page has no more", page: 6, limit: 25, total: 137, expectedTotalPages: 6, expectedHasMore: false},
		{name: "page past the end has no more", page: 99, limit: 10, total: 5, expectedTotalPages: 1, expectedHasMore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := pagination.New(tt.page, tt.limit, tt.total)
			require.Equal(t, tt.page, result.Page)
			require.Equal(t, tt.limit, result.Limit)
			require.Equal(t, tt.total, result.Total)
			require.Equal(t, tt.expectedTotalPages, result.TotalPages)
			require.Equal(t, tt.expectedHasMore, result.HasMore)
		})
	}
}

func TestParseFromRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   string
		expected pagination.Params
	}{
		{
			name:     "absent query parameters use the defaults",
			target:   "/organizations",
			expected: pagination.Params{Page: pagination.DefaultPage, Limit: pagination.DefaultLimit},
		},
		{
			name:     "explicit values are parsed",
			target:   "/organizations?page=3&limit=50",
			expected: pagination.Params{Page: 3, Limit: 50},
		},
		{
			name:     "unparseable page falls back to the default page",
			target:   "/organizations?page=abc",
			expected: pagination.Params{Page: pagination.DefaultPage, Limit: pagination.DefaultLimit},
		},
		{
			name:     "unparseable limit falls back to the default limit",
			target:   "/organizations?page=2&limit=xyz",
			expected: pagination.Params{Page: 2, Limit: pagination.DefaultLimit},
		},
		{
			name:     "out of range values are returned unclamped",
			target:   "/organizations?page=-4&limit=5000",
			expected: pagination.Params{Page: -4, Limit: 5000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			require.Equal(t, tt.expected, pagination.ParseFromRequest(request))
		})
	}
}
