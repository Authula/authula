package util

import "testing"

func TestIsTrustedCallbackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		callbackURL    string
		trustedOrigins []string
		wantErr        string
	}{
		{name: "relative path", callbackURL: "/welcome"},
		{name: "trusted absolute url", callbackURL: "https://app.example.com/welcome", trustedOrigins: []string{"https://app.example.com"}},
		{name: "trusted wildcard subdomain", callbackURL: "https://login.example.com/welcome", trustedOrigins: []string{"https://*.example.com"}},
		{name: "trusted loopback ipv6", callbackURL: "https://[::1]:8443/welcome"},
		{name: "protocol relative", callbackURL: "//evil.example/callback", wantErr: "callback_url must not be protocol-relative"},
		{name: "unsafe scheme", callbackURL: "javascript:alert(1)", wantErr: "callback_url uses an unsafe scheme"},
		{name: "untrusted absolute url", callbackURL: "https://evil.example/callback", trustedOrigins: []string{"https://app.example.com"}, wantErr: "callback_url is not a trusted origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := IsTrustedCallbackURL(tt.callbackURL, tt.trustedOrigins)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestIsTrustedOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		origin         string
		trustedOrigins []string
		want           bool
	}{
		{name: "exact match", origin: "https://app.example.com", trustedOrigins: []string{"https://app.example.com"}, want: true},
		{name: "wildcard subdomain match", origin: "https://login.example.com", trustedOrigins: []string{"https://*.example.com"}, want: true},
		{name: "wildcard base domain match", origin: "https://example.com", trustedOrigins: []string{"https://*.example.com"}, want: true},
		{name: "wildcard domain squatting rejected", origin: "https://malicious-example.com", trustedOrigins: []string{"https://*.example.com"}, want: false},
		{name: "wildcard sibling domain rejected", origin: "https://example.com.evil.com", trustedOrigins: []string{"https://*.example.com"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsTrustedOrigin(tt.origin, tt.trustedOrigins)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
