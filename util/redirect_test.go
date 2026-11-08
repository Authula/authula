package util

import "testing"

func TestValidateRedirectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "https absolute URL", input: "https://api.example.com/auth/oauth2/callback/google"},
		{name: "localhost http", input: "http://localhost:8080/api/auth/oauth2/callback/google"},
		{name: "loopback IPv4 http", input: "http://127.0.0.1:8080/callback"},
		{name: "empty", input: "", wantErr: true},
		{name: "relative path", input: "/callback", wantErr: true},
		{name: "wildcard host", input: "https://*.example.com/callback", wantErr: true},
		{name: "wildcard path", input: "https://example.com/callback/*", wantErr: true},
		{name: "fragment", input: "https://example.com/callback#frag", wantErr: true},
		{name: "credentials", input: "https://user:pass@example.com/callback", wantErr: true},
		{name: "non https remote", input: "http://example.com/callback", wantErr: true},
		{name: "unsafe scheme", input: "javascript:alert(1)", wantErr: true},
		{name: "protocol relative", input: "//example.com/callback", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateRedirectURL(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error for %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

func TestValidateRedirectTo(t *testing.T) {
	t.Parallel()

	trusted := []string{"https://app.example.com", "https://*.example.org"}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty defaults to root", input: "", want: "/"},
		{name: "relative path", input: "/welcome", want: "/welcome"},
		{name: "trusted absolute URL", input: "https://app.example.com/welcome", want: "https://app.example.com/welcome"},
		{name: "trusted wildcard origin", input: "https://login.example.org/welcome", want: "https://login.example.org/welcome"},
		{name: "protocol relative", input: "//evil.example/callback", wantErr: true},
		{name: "javascript scheme", input: "javascript:alert(1)", wantErr: true},
		{name: "credentials", input: "https://user:pass@app.example.com/welcome", wantErr: true},
		{name: "untrusted origin", input: "https://evil.example/callback", wantErr: true},
		{name: "http untrusted origin", input: "http://app.example.com/welcome", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ValidateRedirectTo(tt.input, trusted)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ValidateRedirectTo(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
