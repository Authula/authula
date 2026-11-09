package tests

import coresecurity "github.com/Authula/authula/core/security"

// Test secrets are standard padded Base64 encodings of exactly 32 bytes, the
// only format Authula accepts.
const (
	EncryptionKeyOne        = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	EncryptionKeyTwo        = "ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="
	EncryptionKeyThree      = "YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE="
	EncryptionKeyDevDefault = "YXV0aHVsYS1kZXYta2V5LTAxMjM0NTY3ODlhYmNkZWY="
)

// MustDecodeKey decodes a Base64 test key for tests that need key bytes. It
// panics if the key is invalid so a malformed fixture fails immediately.
func MustDecodeKey(encoded string) []byte {
	key, err := coresecurity.DecodeSecret(encoded)
	if err != nil {
		panic("invalid test key: " + err.Error())
	}

	return key
}
