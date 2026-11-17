package util

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want string
	}{
		{name: "ipv4 masks last octet", ip: "1.2.3.4", want: "1.2.3.x"},
		{name: "loopback masks last octet", ip: "127.0.0.1", want: "127.0.0.x"},
		{name: "empty stays empty", ip: "", want: ""},
		{name: "ipv6 unchanged", ip: "2001:db8::1", want: "2001:db8::1"},
		{name: "ipv4 mapped ipv6 unchanged", ip: "::ffff:1.2.3.4", want: "::ffff:1.2.3.4"},
		{name: "too few octets unchanged", ip: "1.2.3", want: "1.2.3"},
		{name: "too many octets unchanged", ip: "1.2.3.4.5", want: "1.2.3.4.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, MaskIP(tt.ip))
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	t.Parallel()

	trustedProxy := []string{"10.0.0.1"}

	tests := []struct {
		name           string
		remoteAddr     string
		headers        map[string]string
		trustedHeaders []string
		trustedProxies []string
		want           string
		wantErr        bool
	}{
		{
			name:       "uses remote address without trusted proxies",
			remoteAddr: "1.2.3.4:1234",
			headers:    map[string]string{"X-Forwarded-For": "9.9.9.9"},
			want:       "1.2.3.4",
		},
		{
			name:       "accepts remote address without port",
			remoteAddr: "1.2.3.4",
			want:       "1.2.3.4",
		},
		{
			name:       "accepts ipv6 remote address",
			remoteAddr: "[2001:db8::1]:443",
			want:       "2001:db8::1",
		},
		{
			name:       "invalid remote address returns error",
			remoteAddr: "not-an-ip:1234",
			wantErr:    true,
		},
		{
			name:       "invalid remote address without port returns error",
			remoteAddr: "not-an-ip",
			wantErr:    true,
		},
		{
			name:           "untrusted caller ignores headers",
			remoteAddr:     "1.2.3.4:1234",
			headers:        map[string]string{"X-Forwarded-For": "9.9.9.9"},
			trustedProxies: trustedProxy,
			want:           "1.2.3.4",
		},
		{
			name:           "trusted proxy uses forwarded for",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"X-Forwarded-For": "9.9.9.9"},
			trustedProxies: trustedProxy,
			want:           "9.9.9.9",
		},
		{
			name:           "trusted proxy matches cidr",
			remoteAddr:     "10.1.2.3:1234",
			headers:        map[string]string{"X-Forwarded-For": "9.9.9.9"},
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "9.9.9.9",
		},
		{
			name:           "first entry of forwarded list wins",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"X-Forwarded-For": "9.9.9.9, 8.8.8.8"},
			trustedProxies: trustedProxy,
			want:           "9.9.9.9",
		},
		{
			name:           "forwarded entry with port is parsed",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"X-Forwarded-For": "9.9.9.9:8080"},
			trustedProxies: trustedProxy,
			want:           "9.9.9.9",
		},
		{
			name:           "invalid forwarded entries are skipped",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"X-Forwarded-For": "unknown, 9.9.9.9"},
			trustedProxies: trustedProxy,
			want:           "9.9.9.9",
		},
		{
			name:           "no valid forwarded ip falls back to remote",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"X-Forwarded-For": "unknown"},
			trustedProxies: trustedProxy,
			want:           "10.0.0.1",
		},
		{
			name:           "custom trusted header takes priority",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"CF-Connecting-IP": "9.9.9.9", "X-Forwarded-For": "8.8.8.8"},
			trustedHeaders: []string{"CF-Connecting-IP"},
			trustedProxies: trustedProxy,
			want:           "9.9.9.9",
		},
		{
			name:           "empty custom header falls back to forwarded for",
			remoteAddr:     "10.0.0.1:1234",
			headers:        map[string]string{"CF-Connecting-IP": "", "X-Forwarded-For": "8.8.8.8"},
			trustedHeaders: []string{"CF-Connecting-IP"},
			trustedProxies: trustedProxy,
			want:           "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				request.Header.Set(key, value)
			}

			got, err := ExtractClientIP(nil, request, tt.trustedHeaders, tt.trustedProxies)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestGetHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{name: "host and port", remoteAddr: "1.2.3.4:8080", want: "1.2.3.4"},
		{name: "host only", remoteAddr: "1.2.3.4", want: "1.2.3.4"},
		{name: "ipv6 host and port", remoteAddr: "[::1]:8080", want: "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, getHost(tt.remoteAddr))
		})
	}
}

func TestIsIPTrusted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ip      string
		trusted []string
		want    bool
	}{
		{name: "no trusted proxies", ip: "1.2.3.4", trusted: nil, want: false},
		{name: "exact ip match", ip: "1.2.3.4", trusted: []string{"1.2.3.4"}, want: true},
		{name: "cidr match", ip: "10.1.2.3", trusted: []string{"10.0.0.0/8"}, want: true},
		{name: "cidr mismatch", ip: "11.1.2.3", trusted: []string{"10.0.0.0/8"}, want: false},
		{name: "invalid entry is ignored", ip: "1.2.3.4", trusted: []string{"not-a-cidr"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)

			assert.Equal(t, tt.want, isIPTrusted(ip, tt.trusted))
		})
	}
}
