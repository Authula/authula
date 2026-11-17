package util

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildActionURL(t *testing.T) {
	t.Parallel()

	callback := "https://app.example.com/welcome"

	tests := []struct {
		name        string
		baseURL     string
		basePath    string
		actionPath  string
		token       string
		callbackURL *string
		want        string
	}{
		{
			name:       "base path and action path",
			baseURL:    "https://auth.example.com",
			basePath:   "/api/auth",
			actionPath: "/verify-email",
			token:      "abc",
			want:       "https://auth.example.com/api/auth/verify-email?token=abc",
		},
		{
			name:       "no base path",
			baseURL:    "https://auth.example.com",
			actionPath: "/verify-email",
			token:      "abc",
			want:       "https://auth.example.com/verify-email?token=abc",
		},
		{
			name:       "trims surrounding slashes",
			baseURL:    "https://auth.example.com/",
			basePath:   "api/auth/",
			actionPath: "verify-email",
			token:      "abc",
			want:       "https://auth.example.com/api/auth/verify-email?token=abc",
		},
		{
			name:    "no paths",
			baseURL: "https://auth.example.com",
			token:   "abc",
			want:    "https://auth.example.com?token=abc",
		},
		{
			name:        "includes callback url",
			baseURL:     "https://auth.example.com",
			basePath:    "/api/auth",
			actionPath:  "/reset-password",
			token:       "abc",
			callbackURL: &callback,
			want:        "https://auth.example.com/api/auth/reset-password?callback_url=https%3A%2F%2Fapp.example.com%2Fwelcome&token=abc",
		},
		{
			name:        "empty callback is omitted",
			baseURL:     "https://auth.example.com",
			actionPath:  "/reset-password",
			token:       "abc",
			callbackURL: new(""),
			want:        "https://auth.example.com/reset-password?token=abc",
		},
		{
			name:       "token is url encoded",
			baseURL:    "https://auth.example.com",
			actionPath: "/verify",
			token:      "a b&c",
			want:       "https://auth.example.com/verify?token=a+b%26c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildActionURL(tt.baseURL, tt.basePath, tt.actionPath, tt.token, tt.callbackURL)

			parsed, err := url.Parse(got)
			require.NoError(t, err)
			wantParsed, err := url.Parse(tt.want)
			require.NoError(t, err)
			assert.Equal(t, wantParsed, parsed)
		})
	}
}
