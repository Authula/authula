package config

import (
	"testing"
	"time"

	"github.com/Authula/authula/models"
)

type customLogger struct{}

func (l *customLogger) Debug(msg string, args ...any) {}
func (l *customLogger) Info(msg string, args ...any)  {}
func (l *customLogger) Warn(msg string, args ...any)  {}
func (l *customLogger) Error(msg string, args ...any) {}

func TestWithLoggerConfiguresLevelAndCustomLogger(t *testing.T) {
	t.Parallel()

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

func TestWithSessionCookieMaxAge(t *testing.T) {
	tests := []struct {
		name             string
		session          models.SessionConfig
		wantCookieMaxAge time.Duration
	}{
		{
			name:             "defaults keep the cookie alive for the whole session lifetime",
			session:          models.SessionConfig{},
			wantCookieMaxAge: time.Hour * 24 * 7,
		},
		{
			name:             "follows a custom expires_in when cookie_max_age is unset",
			session:          models.SessionConfig{ExpiresIn: 2 * time.Hour},
			wantCookieMaxAge: 2 * time.Hour,
		},
		{
			name:             "explicit cookie_max_age wins over expires_in",
			session:          models.SessionConfig{ExpiresIn: 2 * time.Hour, CookieMaxAge: time.Hour},
			wantCookieMaxAge: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := NewConfig(WithSession(tt.session))

			if config.Session.CookieMaxAge != tt.wantCookieMaxAge {
				t.Fatalf("expected cookie max age %s, got %s", tt.wantCookieMaxAge, config.Session.CookieMaxAge)
			}
		})
	}
}
