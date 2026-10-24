package models

// Logger is the only logging contract used by core and plugins. Implementations
// are interchangeable; slog is merely the default one.
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

// LoggerFanout is an optional capability of a Logger that allows plugins to
// attach additional destinations. Detect it with a type assertion:
//
//	if fanout, ok := logger.(models.LoggerFanout); ok {
//		detach := fanout.AddLogger(sink)
//		defer detach()
//	}
//
// InitLogger always returns a fan-out, including around a user-supplied Logger,
// so plugins can rely on the capability in normal operation.
type LoggerFanout interface {
	// AddLogger attaches sink and returns a function that detaches it. The
	// detach function is safe to call more than once. Nil sinks are ignored.
	AddLogger(sink Logger) (detach func())
}
