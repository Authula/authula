package util

// HashWithSalt returns the hex SHA-256 hash of value and salt joined by a pipe
// separator.
func HashWithSalt(value, salt string) string {
	return SHA256Hex(value + "|" + salt)
}
