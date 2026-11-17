package util

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetQueryString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		key          string
		defaultValue string
		want         string
	}{
		{name: "returns value", url: "/?search=users", key: "search", defaultValue: "default", want: "users"},
		{name: "missing key returns default", url: "/", key: "search", defaultValue: "default", want: "default"},
		{name: "empty value returns default", url: "/?search=", key: "search", defaultValue: "default", want: "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, tt.url, nil)

			assert.Equal(t, tt.want, GetQueryString(request, tt.key, tt.defaultValue))
		})
	}
}

func TestGetQueryInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		key          string
		defaultValue int
		want         int
	}{
		{name: "returns parsed int", url: "/?limit=25", key: "limit", defaultValue: 10, want: 25},
		{name: "negative int is parsed", url: "/?limit=-5", key: "limit", defaultValue: 10, want: -5},
		{name: "missing key returns default", url: "/", key: "limit", defaultValue: 10, want: 10},
		{name: "empty value returns default", url: "/?limit=", key: "limit", defaultValue: 10, want: 10},
		{name: "invalid value returns default", url: "/?limit=abc", key: "limit", defaultValue: 10, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, tt.url, nil)

			assert.Equal(t, tt.want, GetQueryInt(request, tt.key, tt.defaultValue))
		})
	}
}
