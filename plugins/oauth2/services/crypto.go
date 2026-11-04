package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DeriveOAuthHMACKey derives an HMAC key from the app secret for OAuth2 operations
func DeriveOAuthHMACKey(appSecret string) []byte {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte("authula:oauth2:v1"))
	return mac.Sum(nil)
}

// GenerateRandomString generates a cryptographically secure random URL-safe string
// of exactly length characters.
func GenerateRandomString(length int) (string, error) {
	if length < 0 {
		return "", fmt.Errorf("length must not be negative")
	}
	if length == 0 {
		return "", nil
	}

	// Raw URL encoding produces ceil(4*n/3) characters for n bytes.
	byteLen := (length*3 + 3) / 4
	randomBytes := make([]byte, byteLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes)[:length], nil
}

// SignCookie creates a signed cookie value with format: base64(payload).timestamp.signature
func SignCookie(payload string, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("signing secret is empty")
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	data := payload + "|" + timestamp
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s.%s",
		base64.RawURLEncoding.EncodeToString([]byte(payload)),
		timestamp,
		signature), nil
}

// ValidateCookie validates a signed cookie and returns the payload.
func ValidateCookie(signed string, secret []byte, maxAge time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("signing secret is empty")
	}
	if maxAge <= 0 {
		return "", fmt.Errorf("invalid max age")
	}

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

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || ts <= 0 {
		return "", fmt.Errorf("invalid timestamp")
	}

	now := time.Now().Unix()
	if ts > now {
		return "", fmt.Errorf("cookie timestamp is in the future")
	}
	if now-ts > int64(maxAge.Seconds()) {
		return "", fmt.Errorf("cookie expired")
	}

	data := string(payload) + "|" + timestamp
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", fmt.Errorf("signature verification failed")
	}

	return string(payload), nil
}

// GeneratePKCEVerifier generates a PKCE code_verifier.
func GeneratePKCEVerifier() (string, error) {
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", fmt.Errorf("failed to generate verifier: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(verifierBytes), nil
}
