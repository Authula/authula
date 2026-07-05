package bootstrap

import "testing"

type customLogger struct{}

func (l *customLogger) Debug(msg string, args ...any) {}
func (l *customLogger) Info(msg string, args ...any)  {}
func (l *customLogger) Warn(msg string, args ...any)  {}
func (l *customLogger) Error(msg string, args ...any) {}

func TestInitLoggerUsesConfiguredLogger(t *testing.T) {
	logger := &customLogger{}

	if got := InitLogger(LoggerOptions{Level: "debug", Logger: logger}); got != logger {
		t.Fatal("expected configured logger to be used")
	}
}
