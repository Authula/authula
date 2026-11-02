package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/Authula/authula/models"
)

// bridgeLogger adapts models.Logger to a slog-backed otelslog handler. It is
// the only place in the plugin where slog types are used.
type bridgeLogger struct {
	logger *slog.Logger
}

// newBridgeLogger builds a models.Logger that emits records to the supplied
// provider. The level filter mirrors the core logger level so the sink never
// exports records the base logger would have dropped.
func newBridgeLogger(provider *sdklog.LoggerProvider, version string, min slog.Level) models.Logger {
	handler := otelslog.NewHandler(scopeName,
		otelslog.WithLoggerProvider(provider),
		otelslog.WithVersion(version),
	)

	return &bridgeLogger{logger: slog.New(levelHandler{
		min:   min,
		inner: handler,
	})}
}

func (b *bridgeLogger) Debug(msg string, args ...any) { b.logger.Debug(msg, args...) }
func (b *bridgeLogger) Info(msg string, args ...any)  { b.logger.Info(msg, args...) }
func (b *bridgeLogger) Warn(msg string, args ...any)  { b.logger.Warn(msg, args...) }
func (b *bridgeLogger) Error(msg string, args ...any) { b.logger.Error(msg, args...) }

// levelHandler enforces the core logger level on the OTel sink.
type levelHandler struct {
	min   slog.Level
	inner slog.Handler
}

func (h levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && h.inner.Enabled(ctx, level)
}

func (h levelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.inner.Handle(ctx, record)
}

func (h levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelHandler{min: h.min, inner: h.inner.WithAttrs(attrs)}
}

func (h levelHandler) WithGroup(name string) slog.Handler {
	return levelHandler{min: h.min, inner: h.inner.WithGroup(name)}
}

func parseLevel(name string) slog.Level {
	switch name {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
