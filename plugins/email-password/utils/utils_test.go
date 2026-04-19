package utils

import (
	"testing"
)

func TestBuildVerificationURL(t *testing.T) {
	callbackWithValue := "https://app.com/callback?token=abc123"
	emptyCallback := ""

	tests := []struct {
		name     string
		baseURL  string
		basePath string
		token    string
		callback *string
		want     string
	}{
		{
			name:     "with callback",
			baseURL:  "https://example.com",
			basePath: "/auth",
			token:    "abc123",
			callback: &callbackWithValue,
			want:     "https://example.com/auth/email-password/verify-email?callback_url=https%3A%2F%2Fapp.com%2Fcallback%3Ftoken%3Dabc123&token=abc123",
		},
		{
			name:     "without callback",
			baseURL:  "https://example.com",
			basePath: "/auth",
			token:    "xyz789",
			callback: nil,
			want:     "https://example.com/auth/email-password/verify-email?token=xyz789",
		},
		{
			name:     "with empty callback",
			baseURL:  "https://example.com",
			basePath: "/auth",
			token:    "token",
			callback: &emptyCallback,
			want:     "https://example.com/auth/email-password/verify-email?token=token",
		},
		{
			name:     "with trailing slash in base path",
			baseURL:  "https://example.com",
			basePath: "/auth/",
			token:    "token",
			callback: nil,
			want:     "https://example.com/auth/email-password/verify-email?token=token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildVerificationURL(tt.baseURL, tt.basePath, tt.token, tt.callback)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
