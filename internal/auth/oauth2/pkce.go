package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
)

// GeneratePKCE generates a code verifier and code challenge for PKCE.
func GeneratePKCE() (verifier string, challenge string, err error) {
	// Generate random verifier (32 bytes -> 43 chars base64url)
	// RFC 7636 recommends 43-128 chars. 32 bytes of entropy is standard.
	verifier, err = util.GenerateRandomTokenBase64URL(32)
	if err != nil {
		return "", "", fmt.Errorf("generate verifier: %w", err)
	}

	// Calculate S256 challenge
	// challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
	h := sha256.New()
	h.Write([]byte(verifier))
	sum := h.Sum(nil)
	challenge = base64.RawURLEncoding.EncodeToString(sum)

	return verifier, challenge, nil
}
