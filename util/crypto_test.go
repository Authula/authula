package util

import (
	"encoding/hex"
	"testing"
)

func TestHashWithSalt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		salt  string
		want  string
	}{
		{
			name:  "known pair",
			value: "value-1",
			salt:  "salt-1",
			want:  "59925c49b13f4d9d0d6583f5146843f8baa478dc6b4849754c7f89bc93d4324a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := HashWithSalt(tt.value, tt.salt)
			if got != tt.want {
				t.Fatalf("HashWithSalt(%q, %q) = %q, want %q", tt.value, tt.salt, got, tt.want)
			}
		})
	}
}

func TestHashWithSalt_ChangesWithSalt(t *testing.T) {
	t.Parallel()

	first := HashWithSalt("value-1", "salt-1")
	second := HashWithSalt("value-1", "salt-2")

	if first == second {
		t.Fatal("expected different salts to produce different hashes")
	}
	if len(first) != 64 {
		t.Fatalf("expected a 64-character hash, got %d characters", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("hash is not valid hex: %v", err)
	}
}
