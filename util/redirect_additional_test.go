package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRedirectURL_AdditionalCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "whitespace only", input: "   ", wantErr: "redirect_url is required"},
		{name: "unsupported scheme", input: "ftp://example.com/callback", wantErr: "redirect_url must use http or https"},
		{name: "invalid url", input: "https://[::1", wantErr: "redirect_url must be a valid absolute URL"},
		{name: "trailing fragment marker", input: "https://example.com/callback#", wantErr: "redirect_url must not contain a fragment"},
		{name: "query string allowed", input: "https://example.com/callback?source=login"},
		{name: "surrounding whitespace is trimmed", input: "  https://example.com/callback  "},
		{name: "https localhost", input: "https://localhost:8443/callback"},
		{name: "ipv6 loopback http", input: "http://[::1]:8080/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateRedirectURL(tt.input)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
