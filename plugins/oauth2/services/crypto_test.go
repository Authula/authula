package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()

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
		})
	}
}

func TestGeneratePKCEVerifier(t *testing.T) {
	t.Parallel()

	const base64URLAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	seen := make(map[string]struct{}, 10)
	for range 10 {
		verifier, err := GeneratePKCEVerifier()
		if err != nil {
			t.Fatalf("GeneratePKCEVerifier returned error: %v", err)
		}
		if len(verifier) != 43 {
			t.Fatalf("expected verifier length 43, got %d", len(verifier))
		}
		if strings.IndexFunc(verifier, func(r rune) bool { return !strings.ContainsRune(base64URLAlphabet, r) }) != -1 {
			t.Fatalf("verifier contains non-base64url characters: %q", verifier)
		}
		if _, exists := seen[verifier]; exists {
			t.Fatalf("GeneratePKCEVerifier returned a duplicate verifier: %q", verifier)
		}
		seen[verifier] = struct{}{}

		challenge := oauth2.S256ChallengeFromVerifier(verifier)
		if len(challenge) != 43 {
			t.Fatalf("expected challenge length 43, got %d", len(challenge))
		}
	}

	if len(seen) != 10 {
		t.Fatalf("expected 10 unique verifiers, got %d", len(seen))
	}
}

func TestValidateCookieRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	valid, err := SignCookie("payload", secret)
	if err != nil {
		t.Fatalf("SignCookie failed: %v", err)
	}

	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected cookie format: %s", valid)
	}

	tests := []struct {
		name   string
		cookie string
		maxAge time.Duration
	}{
		{name: "tampered signature", cookie: parts[0] + "." + parts[1] + ".deadbeef", maxAge: 5 * time.Minute},
		{name: "zero timestamp", cookie: parts[0] + ".0." + parts[2], maxAge: 5 * time.Minute},
		{name: "partial timestamp", cookie: parts[0] + "." + parts[1] + "abc." + parts[2], maxAge: 5 * time.Minute},
		{name: "future timestamp", cookie: signedCookieWithTimestamp("payload", secret, time.Now().Add(time.Hour).Unix()), maxAge: 5 * time.Minute},
		{name: "expired", cookie: signedCookieWithTimestamp("payload", secret, time.Now().Add(-time.Hour).Unix()), maxAge: 5 * time.Minute},
		{name: "wrong segment count", cookie: "not-a-cookie", maxAge: 5 * time.Minute},
		{name: "zero max age", cookie: valid, maxAge: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := ValidateCookie(tt.cookie, secret, tt.maxAge); err == nil {
				t.Fatalf("expected an error for %s", tt.name)
			}
		})
	}

	if got, err := ValidateCookie(valid, secret, 5*time.Minute); err != nil || got != "payload" {
		t.Fatalf("valid cookie rejected: got %q, err %v", got, err)
	}
}

func signedCookieWithTimestamp(payload string, secret []byte, timestamp int64) string {
	encoded := strings.Split(mustSign(payload, secret), ".")[0]
	data := fmt.Sprintf("%s|%d", payload, timestamp)
	return fmt.Sprintf("%s.%d.%s", encoded, timestamp, hmacHex(data, secret))
}

func mustSign(payload string, secret []byte) string {
	signed, err := SignCookie(payload, secret)
	if err != nil {
		panic(err)
	}
	return signed
}

func hmacHex(data string, secret []byte) string {
	// Mirrors the signature format used by SignCookie.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
