package config

import (
	"testing"
	"time"

	"github.com/Authula/authula/env"
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

func TestWithSessionCookieDomain(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		domain     string
		wantDomain string
		wantPanic  bool
	}{
		{
			name:       "empty domain keeps cookies host-only",
			baseURL:    "http://localhost:8080",
			domain:     "",
			wantDomain: "",
		},
		{
			name:       "parent domain of the base url host is accepted",
			baseURL:    "https://api.example.com",
			domain:     "example.com",
			wantDomain: "example.com",
		},
		{
			name:       "domain equal to the base url host is accepted",
			baseURL:    "https://example.com",
			domain:     "example.com",
			wantDomain: "example.com",
		},
		{
			name:       "leading dot is stripped",
			baseURL:    "https://api.example.com",
			domain:     ".example.com",
			wantDomain: "example.com",
		},
		{
			name:       "domain is lower-cased",
			baseURL:    "https://api.example.com:8443",
			domain:     "Example.COM",
			wantDomain: "example.com",
		},
		{
			name:      "base url host outside the domain panics",
			baseURL:   "http://localhost:8080",
			domain:    "example.com",
			wantPanic: true,
		},
		{
			name:      "look-alike suffix without a dot boundary panics",
			baseURL:   "https://notexample.com",
			domain:    "example.com",
			wantPanic: true,
		},
		{
			name:      "domain with a scheme panics",
			baseURL:   "https://api.example.com",
			domain:    "https://example.com",
			wantPanic: true,
		},
		{
			name:      "domain with a port panics",
			baseURL:   "https://api.example.com",
			domain:    "example.com:8080",
			wantPanic: true,
		},
		{
			name:      "bare dot panics",
			baseURL:   "https://api.example.com",
			domain:    ".",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				recovered := recover()
				if tt.wantPanic && recovered == nil {
					t.Fatal("expected NewConfig to panic")
				}
				if !tt.wantPanic && recovered != nil {
					t.Fatalf("expected NewConfig not to panic, got %v", recovered)
				}
			}()

			config := NewConfig(
				WithBaseURL(tt.baseURL),
				WithSession(models.SessionConfig{Domain: tt.domain}),
			)

			if config.Session.Domain != tt.wantDomain {
				t.Fatalf("expected cookie domain %q, got %q", tt.wantDomain, config.Session.Domain)
			}
		})
	}
}

// Uses t.Setenv, so it must not run in parallel with the other config tests.
func TestWithSessionCookieDomainFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envDomain    string
		configDomain string
		baseURL      string
		wantDomain   string
		wantPanic    bool
	}{
		{
			name:       "env var sets the domain when config leaves it empty",
			envDomain:  "example.com",
			baseURL:    "https://api.example.com",
			wantDomain: "example.com",
		},
		{
			name:         "env var overrides the domain from config",
			envDomain:    "example.com",
			configDomain: "other.com",
			baseURL:      "https://api.example.com",
			wantDomain:   "example.com",
		},
		{
			name:         "config domain is used when env var is unset",
			envDomain:    "",
			configDomain: "example.com",
			baseURL:      "https://api.example.com",
			wantDomain:   "example.com",
		},
		{
			name:       "env var is normalized like config",
			envDomain:  ".Example.COM",
			baseURL:    "https://api.example.com",
			wantDomain: "example.com",
		},
		{
			name:      "env var is validated against base url",
			envDomain: "example.com",
			baseURL:   "http://localhost:8080",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvCookieDomain, tt.envDomain)

			defer func() {
				recovered := recover()
				if tt.wantPanic && recovered == nil {
					t.Fatal("expected NewConfig to panic")
				}
				if !tt.wantPanic && recovered != nil {
					t.Fatalf("expected NewConfig not to panic, got %v", recovered)
				}
			}()

			config := NewConfig(
				WithBaseURL(tt.baseURL),
				WithSession(models.SessionConfig{Domain: tt.configDomain}),
			)

			if config.Session.Domain != tt.wantDomain {
				t.Fatalf("expected cookie domain %q, got %q", tt.wantDomain, config.Session.Domain)
			}
		})
	}
}

// Standalone mode builds the config without WithSession, so the env var must still apply.
func TestNewConfigCookieDomainFromEnvWithoutSessionOption(t *testing.T) {
	t.Setenv(env.EnvCookieDomain, "example.com")
	t.Setenv(env.EnvBaseURL, "https://api.example.com")

	config := NewConfig()

	if config.Session.Domain != "example.com" {
		t.Fatalf("expected cookie domain %q, got %q", "example.com", config.Session.Domain)
	}
}
