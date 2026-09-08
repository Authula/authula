package sqlitedsn_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Authula/authula/internal/sqlitedsn"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		wantBase   string
		wantParams map[string]string
	}{
		{
			name:       "in memory target",
			target:     ":memory:",
			wantBase:   "file::memory:",
			wantParams: map[string]string{"_busy_timeout": "5000", "_timezone": "UTC"},
		},
		{
			name:       "relative file path",
			target:     "./data/authula.db",
			wantBase:   "file:./data/authula.db",
			wantParams: map[string]string{"_busy_timeout": "5000", "_timezone": "UTC"},
		},
		{
			name:       "existing file uri keeps its parameters",
			target:     "file:shared.db?mode=memory&cache=shared",
			wantBase:   "file:shared.db",
			wantParams: map[string]string{"mode": "memory", "cache": "shared", "_busy_timeout": "5000", "_timezone": "UTC"},
		},
		{
			name:       "caller parameters win over defaults",
			target:     "file:authula.db?_busy_timeout=250&_timezone=Local",
			wantBase:   "file:authula.db",
			wantParams: map[string]string{"_busy_timeout": "250", "_timezone": "Local"},
		},
		{
			name:       "driver alias suppresses the default it duplicates",
			target:     "file:authula.db?_timeout=250",
			wantBase:   "file:authula.db",
			wantParams: map[string]string{"_timeout": "250", "_timezone": "UTC"},
		},
		{
			name:       "path delimiters are escaped",
			target:     "./data/rate#1.db",
			wantBase:   "file:./data/rate%231.db",
			wantParams: map[string]string{"_busy_timeout": "5000", "_timezone": "UTC"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := sqlitedsn.Build(tc.target)

			base, query, found := strings.Cut(got, "?")
			if !found {
				t.Fatalf("expected query parameters in %q", got)
			}
			if base != tc.wantBase {
				t.Fatalf("expected base %q, got %q", tc.wantBase, base)
			}

			params, err := url.ParseQuery(query)
			if err != nil {
				t.Fatalf("unexpected error parsing %q: %v", query, err)
			}
			if len(params) != len(tc.wantParams) {
				t.Fatalf("expected %d parameters, got %d in %q", len(tc.wantParams), len(params), query)
			}
			for key, want := range tc.wantParams {
				if got := params.Get(key); got != want {
					t.Fatalf("expected %s=%q, got %q", key, want, got)
				}
			}
		})
	}
}

func TestBuildLeavesUnparsableQueryUntouched(t *testing.T) {
	t.Parallel()

	target := "file:authula.db?%zz"

	if got := sqlitedsn.Build(target); got != target {
		t.Fatalf("expected %q to be returned unchanged, got %q", target, got)
	}
}
