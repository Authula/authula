package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DeriveOAuthHMACKey derives an HMAC key from the app secret for OAuth2 operations
func DeriveOAuthHMACKey(appSecret string) []byte {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte("gobetterauth:oauth2:v1"))
	return mac.Sum(nil)
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// SignCookie creates a signed cookie value with format: payload.timestamp.signature
func SignCookie(payload string, secret []byte) (string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Create signature: HMAC(payload|timestamp)
	data := fmt.Sprintf("%s|%s", payload, timestamp)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Format: base64(payload).timestamp.signature
	encoded := fmt.Sprintf("%s.%s.%s",
		base64.RawURLEncoding.EncodeToString([]byte(payload)),
		timestamp,
		signature)

	return encoded, nil
}

// ValidateCookie validates a signed cookie and returns the payload
func ValidateCookie(signed string, secret []byte, maxAge time.Duration) (string, error) {
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid cookie format")
	}

	encodedPayload := parts[0]
	timestamp := parts[1]
	signature := parts[2]

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", fmt.Errorf("invalid payload encoding: %w", err)
	}

	var ts int64
	_, err = fmt.Sscanf(timestamp, "%d", &ts)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp: %w", err)
	}

	// Check expiry
	if time.Since(time.Unix(ts, 0)) > maxAge {
		return "", fmt.Errorf("cookie expired")
	}

	// Verify signature
	data := fmt.Sprintf("%s|%s", string(payload), timestamp)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", fmt.Errorf("signature verification failed")
	}

	return string(payload), nil
}

// GeneratePKCEChallenge generates PKCE code_verifier and code_challenge
func GeneratePKCEChallenge() (verifier string, challenge string, err error) {
	// Generate code_verifier (32 bytes for maximum entropy)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code_challenge = base64url(sha256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}
