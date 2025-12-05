package domain

import (
	"net/http"
	"os"
	"time"
)

// =======================
// Database Config
// =======================

type DatabaseConfig struct {
	Provider         string
	ConnectionString string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

// =======================
// Email/Password Auth Config
// =======================

type EmailPasswordConfig struct {
	Enabled                  bool
	MinPasswordLength        int
	MaxPasswordLength        int
	DisableSignUp            bool
	RequireEmailVerification bool
	AutoSignIn               bool
	SendResetPasswordEmail   func(user User, url string, token string) error
	ResetTokenExpiry         time.Duration
	Password                 *PasswordConfig
}

// =======================
// Password Config
// =======================

type PasswordConfig struct {
	Hash   func(password string) (string, error)
	Verify func(hashedPassword, password string) bool
}

// =======================
// Email Verification Config
// =======================

type EmailVerificationConfig struct {
	SendVerificationEmail func(user User, url string, token string) error
	AutoSignIn            bool
	SendOnSignUp          bool
	SendOnSignIn          bool
	ExpiresIn             time.Duration
}

// =======================
// User Config
// =======================

type ChangeEmailConfig struct {
	Enabled                          bool
	SendEmailChangeVerificationEmail func(user User, newEmail string, url string, token string) error
}

type UserConfig struct {
	ChangeEmail ChangeEmailConfig
}

// =======================
// Session Config
// =======================

type SessionConfig struct {
	ExpiresIn  time.Duration
	UpdateAge  time.Duration
	CookieName string
}

// =======================
// Social Providers Config
// =======================

type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type DefaultOAuth2ProvidersConfig struct {
	Google  *OAuth2Config
	GitHub  *OAuth2Config
	Discord *OAuth2Config
}

type GenericOAuth2EndpointConfig struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}

type GenericOAuth2Config struct {
	OAuth2Config
	Endpoint GenericOAuth2EndpointConfig
}

type SocialProvidersConfig struct {
	Default DefaultOAuth2ProvidersConfig
	Generic map[string]GenericOAuth2Config
}

// =======================
// Trusted Origins Config
// =======================

type TrustedOriginsConfig struct {
	Origins []string
}

// =======================
// Endpoint Hooks Config
// =======================

type EndpointHookContext struct {
	Path    string
	Method  string
	Body    map[string]any
	Headers map[string]string
	Query   map[string]string
	Request *http.Request
	User    *User
}

type EndpointHooksConfig struct {
	Before func(ctx *EndpointHookContext) error
	After  func(ctx *EndpointHookContext) error
}

// =======================
// Database Hooks Config
// =======================

type DatabaseHooksConfig struct {
	Users         *UserDatabaseHooksConfig
	Accounts      *AccountDatabaseHooksConfig
	Sessions      *SessionDatabaseHooksConfig
	Verifications *VerificationDatabaseHooksConfig
}

type UserDatabaseHooksConfig struct {
	BeforeCreate func(user *User) error
	AfterCreate  func(user User) error
	BeforeUpdate func(user *User) error
	AfterUpdate  func(user User) error
}

type AccountDatabaseHooksConfig struct {
	BeforeCreate func(account *Account) error
	AfterCreate  func(account Account) error
	BeforeUpdate func(account *Account) error
	AfterUpdate  func(account Account) error
}

type SessionDatabaseHooksConfig struct {
	BeforeCreate func(session *Session) error
	AfterCreate  func(session Session) error
}

type VerificationDatabaseHooksConfig struct {
	BeforeCreate func(verification *Verification) error
	AfterCreate  func(verification Verification) error
}

// =======================
// Event Hooks Config
// =======================

type EventHooksConfig struct {
	OnUserSignedUp    func(user User) error
	OnUserLoggedIn    func(user User) error
	OnEmailVerified   func(user User) error
	OnPasswordChanged func(user User) error
	OnEmailChanged    func(user User) error
}

// =======================
// Main Config Structure
// =======================

// Config holds all configurable options for the GoBetterAuth library.
type Config struct {
	AppName           string
	BaseURL           string
	BasePath          string
	Secret            string
	Database          DatabaseConfig
	EmailPassword     EmailPasswordConfig
	EmailVerification EmailVerificationConfig
	User              UserConfig
	Session           SessionConfig
	SocialProviders   SocialProvidersConfig
	TrustedOrigins    TrustedOriginsConfig
	EndpointHooks     EndpointHooksConfig
	DatabaseHooks     DatabaseHooksConfig
	EventHooks        EventHooksConfig
}

// =======================
// Functional Options
// =======================

type ConfigOption func(*Config)

// NewConfig builds a Config using functional options with sensible defaults.
func NewConfig(opts ...ConfigOption) *Config {
	baseURL := os.Getenv("GO_BETTER_AUTH_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	secret := os.Getenv("GO_BETTER_AUTH_SECRET")
	if secret == "" {
		env := os.Getenv("GO_ENV")
		// Must be set in production
		if env == "production" {
			panic("GO_BETTER_AUTH_SECRET environment variable must be set in production")
		}
		// Use default secret for non-production environments
		secret = "go-better-auth-secret-0123456789"
	}

	// Define sensible defaults first
	c := &Config{
		AppName:  "GoBetterAuth",
		BaseURL:  baseURL,
		BasePath: "/auth",
		Secret:   secret,
		Database: DatabaseConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
		},
		EmailPassword: EmailPasswordConfig{
			Enabled:                  false,
			RequireEmailVerification: false,
			MinPasswordLength:        8,
			MaxPasswordLength:        32,
		},
		EmailVerification: EmailVerificationConfig{
			AutoSignIn:   false,
			SendOnSignUp: false,
			SendOnSignIn: false,
			ExpiresIn:    1 * time.Hour,
		},
		User: UserConfig{
			ChangeEmail: ChangeEmailConfig{},
		},
		Session: SessionConfig{
			CookieName: "go-better-auth.session_token",
			ExpiresIn:  7 * 24 * time.Hour, // (default: 7 days)
			UpdateAge:  24 * time.Hour,
		},
		TrustedOrigins: TrustedOriginsConfig{
			Origins: []string{},
		},
		EndpointHooks: EndpointHooksConfig{},
		DatabaseHooks: DatabaseHooksConfig{},
		EventHooks:    EventHooksConfig{},
	}

	// Apply the options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// =======================
// Functional Options
// =======================

func WithAppName(name string) ConfigOption {
	return func(c *Config) {
		c.AppName = name
	}
}

func WithBaseURL(url string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = url
	}
}

func WithBasePath(path string) ConfigOption {
	return func(c *Config) {
		c.BasePath = path
	}
}

func WithSecret(secret string) ConfigOption {
	return func(c *Config) {
		c.Secret = secret
	}
}

func WithDatabase(db DatabaseConfig) ConfigOption {
	return func(c *Config) {
		c.Database = db
	}
}

func WithEmailPassword(config EmailPasswordConfig) ConfigOption {
	return func(c *Config) {
		c.EmailPassword = config
	}
}

func WithEmailVerification(config EmailVerificationConfig) ConfigOption {
	return func(c *Config) {
		c.EmailVerification = config
	}
}

func WithUser(userConfig UserConfig) ConfigOption {
	return func(c *Config) {
		c.User = userConfig
	}
}

func WithSession(sessionConfig SessionConfig) ConfigOption {
	return func(c *Config) {
		c.Session = sessionConfig
	}
}

func WithTrustedOrigins(trustedOriginsConfig TrustedOriginsConfig) ConfigOption {
	return func(c *Config) {
		c.TrustedOrigins = trustedOriginsConfig
	}
}

func WithEndpointHooks(endpointHooksConfig EndpointHooksConfig) ConfigOption {
	return func(c *Config) {
		c.EndpointHooks = endpointHooksConfig
	}
}

func WithDatabaseHooks(databaseHooksConfig DatabaseHooksConfig) ConfigOption {
	return func(c *Config) {
		c.DatabaseHooks = databaseHooksConfig
	}
}

func WithEventHooks(eventHooksConfig EventHooksConfig) ConfigOption {
	return func(c *Config) {
		c.EventHooks = eventHooksConfig
	}
}

func WithSocialProviders(socialProvidersConfig SocialProvidersConfig) ConfigOption {
	return func(c *Config) {
		c.SocialProviders = socialProvidersConfig
	}
}
