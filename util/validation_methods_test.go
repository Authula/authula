package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validationTestUser struct {
	Name string `validate:"required"`
}

func TestValidateStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   validationTestUser
		wantErr bool
	}{
		{name: "valid struct", input: validationTestUser{Name: "user"}},
		{name: "missing required field", input: validationTestUser{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateStruct(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, Validate)
		})
	}
}

func TestValidateTrustedOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origins []string
		wantErr string
	}{
		{name: "nil origins", origins: nil},
		{name: "https origin", origins: []string{"https://app.example.com"}},
		{name: "http origin", origins: []string{"http://localhost:3000"}},
		{name: "wildcard origin", origins: []string{"https://*.example.com"}},
		{name: "non standard scheme without localhost", origins: []string{"ftp://example.com"}},
		{
			name:    "missing scheme",
			origins: []string{"app.example.com"},
			wantErr: "must include scheme",
		},
		{
			name:    "missing host",
			origins: []string{"https://"},
			wantErr: "must include host",
		},
		{
			name:    "invalid url",
			origins: []string{"https://[::1"},
			wantErr: "invalid trusted origin",
		},
		{
			name:    "non standard scheme with localhost",
			origins: []string{"ftp://localhost"},
			wantErr: "uses non-standard scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTrustedOrigins(tt.origins)

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestIsTrustedOrigin_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origin  string
		trusted []string
		want    bool
	}{
		{
			name:    "wildcard with matching port",
			origin:  "https://login.example.com:8443",
			trusted: []string{"https://*.example.com:8443"},
			want:    true,
		},
		{
			name:    "wildcard with port mismatch",
			origin:  "https://login.example.com:9443",
			trusted: []string{"https://*.example.com:8443"},
			want:    false,
		},
		{
			name:    "wildcard scheme mismatch",
			origin:  "http://login.example.com",
			trusted: []string{"https://*.example.com"},
			want:    false,
		},
		{
			name:    "exact host with different port",
			origin:  "https://app.example.com:8443",
			trusted: []string{"https://app.example.com"},
			want:    false,
		},
		{
			name:    "invalid origin url",
			origin:  "https://[::1",
			trusted: []string{"https://app.example.com"},
			want:    false,
		},
		{
			name:    "invalid pattern url",
			origin:  "https://app.example.com",
			trusted: []string{"https://[::1"},
			want:    false,
		},
		{
			name:    "empty trusted list",
			origin:  "https://app.example.com",
			trusted: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsTrustedOrigin(tt.origin, tt.trusted))
		})
	}
}

func TestIsTrustedCallbackURL_AdditionalCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		callbackURL    string
		trustedOrigins []string
		wantErr        string
	}{
		{
			name:        "empty",
			callbackURL: "",
			wantErr:     "callback_url is required",
		},
		{
			name:        "opaque https url",
			callbackURL: "https:foo",
			wantErr:     "callback_url must be an absolute URL or relative path",
		},
		{
			name:        "relative path without leading slash",
			callbackURL: "welcome",
			wantErr:     "callback_url must be an absolute URL or relative path",
		},
		{
			name:        "credentials",
			callbackURL: "https://user:pass@app.example.com/welcome",
			wantErr:     "callback_url must not contain credentials",
		},
		{
			name:        "absolute url without trusted origins",
			callbackURL: "https://app.example.com/welcome",
			wantErr:     "no trusted origins configured",
		},
		{
			name:        "http localhost without trusted origins",
			callbackURL: "http://localhost:3000/welcome",
			wantErr:     "no trusted origins configured",
		},
		{
			name:           "relative path with query",
			callbackURL:    "/welcome?next=%2Fhome",
			trustedOrigins: []string{"https://app.example.com"},
		},
		{
			name:           "loopback ipv4 https",
			callbackURL:    "https://127.0.0.1:8443/welcome",
			trustedOrigins: nil,
		},
		{
			name:        "invalid url",
			callbackURL: "https://[::1",
			wantErr:     "invalid callback_url",
		},
		{
			name:        "scheme without host or opaque",
			callbackURL: "https:",
			wantErr:     "callback_url must be an absolute URL or relative path",
		},
		{
			name:        "unsafe scheme with host",
			callbackURL: "ftp://example.com/welcome",
			wantErr:     "callback_url uses an unsafe scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := IsTrustedCallbackURL(tt.callbackURL, tt.trustedOrigins)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost", host: "localhost", want: true},
		{name: "localhost with port", host: "localhost:3000", want: true},
		{name: "loopback ipv4", host: "127.0.0.1", want: true},
		{name: "loopback ipv4 with port", host: "127.0.0.1:3000", want: true},
		{name: "loopback ipv4 range", host: "127.0.0.2", want: true},
		{name: "loopback ipv6", host: "::1", want: true},
		{name: "loopback ipv6 bracketed", host: "[::1]", want: true},
		{name: "loopback ipv6 bracketed with port", host: "[::1]:8080", want: true},
		{name: "remote host", host: "app.example.com", want: false},
		{name: "remote ip", host: "8.8.8.8", want: false},
		{name: "empty", host: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsLocalhost(tt.host))
		})
	}
}

func TestValidateTrustedHeadersAndProxies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		trustedHeaders []string
		trustedProxies []string
		wantWarnings   int
	}{
		{
			name:           "warns when headers are trusted without proxies",
			trustedHeaders: []string{"X-Forwarded-For"},
			wantWarnings:   1,
		},
		{
			name:           "no warning when proxies are configured",
			trustedHeaders: []string{"X-Forwarded-For"},
			trustedProxies: []string{"10.0.0.1"},
		},
		{
			name: "no warning without headers or proxies",
		},
		{
			name:           "no warning without headers",
			trustedProxies: []string{"10.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &recordingLogger{}

			ValidateTrustedHeadersAndProxies(logger, tt.trustedHeaders, tt.trustedProxies)

			records := logger.Records()
			require.Len(t, records, tt.wantWarnings)
			if tt.wantWarnings == 0 {
				return
			}
			assert.Equal(t, "warn", records[0].Level)
			assert.Contains(t, records[0].Message, "TrustedHeaders")
		})
	}
}
