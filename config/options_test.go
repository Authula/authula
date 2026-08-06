package config

import (
	"testing"

	"github.com/Authula/authula/models"
)

type customLogger struct{}

func (l *customLogger) Debug(msg string, args ...any) {}
func (l *customLogger) Info(msg string, args ...any)  {}
func (l *customLogger) Warn(msg string, args ...any)  {}
func (l *customLogger) Error(msg string, args ...any) {}

func TestWithLoggerConfiguresLevelAndCustomLogger(t *testing.T) {
	logger := &customLogger{}

	config := NewConfig(WithLogger(models.LoggerConfig{
		Level:  "debug",
		Logger: logger,
	}))

	if config.Logger.Level != "debug" {
		t.Fatalf("expected logger level debug, got %q", config.Logger.Level)
	}

	if config.Logger.Logger != logger {
		t.Fatal("expected custom logger to be configured")
	}
}
