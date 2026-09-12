package types

import "testing"

func TestParseEmailProviderType(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    EmailProviderType
		wantErr bool
	}{
		{
			name: "parses smtp provider",
			raw:  "smtp",
			want: ProviderSMTP,
		},
		{
			name: "parses resend provider",
			raw:  "resend",
			want: ProviderResend,
		},
		{
			name: "is case insensitive",
			raw:  "ReSeNd",
			want: ProviderResend,
		},
		{
			name: "trims surrounding whitespace",
			raw:  "  smtp\t",
			want: ProviderSMTP,
		},
		{
			name:    "rejects an unsupported provider",
			raw:     "sendgrid",
			wantErr: true,
		},
		{
			name:    "rejects an empty value",
			raw:     "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEmailProviderType(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseEmailProviderType(%q) expected an error, got provider %q", tt.raw, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseEmailProviderType(%q) returned unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseEmailProviderType(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSupportedEmailProviders(t *testing.T) {
	providers := SupportedEmailProviders()

	if len(providers) != 2 {
		t.Fatalf("got %d supported providers, want 2", len(providers))
	}

	for _, provider := range providers {
		if _, err := ParseEmailProviderType(provider.String()); err != nil {
			t.Errorf("supported provider %q is not parseable: %v", provider, err)
		}
	}
}
