package util

import (
	"net/http"
	"testing"
)

func TestParseSameSite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  http.SameSite
	}{
		{name: "lax", value: "lax", want: http.SameSiteLaxMode},
		{name: "strict", value: "strict", want: http.SameSiteStrictMode},
		{name: "none", value: "none", want: http.SameSiteNoneMode},
		{name: "case insensitive", value: "Strict", want: http.SameSiteStrictMode},
		{name: "empty falls back to lax", value: "", want: http.SameSiteLaxMode},
		{name: "unknown falls back to lax", value: "whatever", want: http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ParseSameSite(tt.value); got != tt.want {
				t.Fatalf("ParseSameSite(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
