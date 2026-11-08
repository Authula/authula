package util

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()

	const base64URLAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	for _, length := range []int{0, 1, 32, 43, 44, 64, 100} {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			t.Parallel()

			got, err := GenerateRandomString(length)
			if err != nil {
				t.Fatalf("GenerateRandomString(%d) returned error: %v", length, err)
			}
			if len(got) != length {
				t.Fatalf("GenerateRandomString(%d) returned length %d", length, len(got))
			}
			if strings.IndexFunc(got, func(r rune) bool { return !strings.ContainsRune(base64URLAlphabet, r) }) != -1 {
				t.Fatalf("GenerateRandomString(%d) returned non-base64url characters: %q", length, got)
			}
		})
	}
}

func TestGenerateRandomString_Unique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 10)
	for range 10 {
		value, err := GenerateRandomString(43)
		if err != nil {
			t.Fatalf("GenerateRandomString returned error: %v", err)
		}
		if _, exists := seen[value]; exists {
			t.Fatalf("GenerateRandomString returned a duplicate value: %q", value)
		}
		seen[value] = struct{}{}
	}
}
