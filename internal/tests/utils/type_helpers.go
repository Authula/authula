package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"uuid"
)

// GenerateHashedToken creates a secure random 256-bit token,
// hashes it with SHA-256, and returns it as a hex string.
func GenerateHashedToken() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("failed to read random bytes for test token: " + err.Error())
	}

	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Generates a UUID
func GenerateUUID() uuid.UUID {
	return uuid.New()
}
