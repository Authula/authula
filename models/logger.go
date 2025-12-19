package models

// Logger defines an interface for logging operations, allowing users to plug in
// different logging implementations such as slog, zerolog, or others.
type Logger interface {
	// Debug logs a message at debug level with optional key-value pairs.
	Debug(msg string, args ...any)
	// Info logs a message at info level with optional key-value pairs.
	Info(msg string, args ...any)
	// Warn logs a message at warn level with optional key-value pairs.
	Warn(msg string, args ...any)
	// Error logs a message at error level with optional key-value pairs.
	Error(msg string, args ...any)
}
