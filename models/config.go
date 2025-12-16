package models

import (
	"net/http"
	"time"

	"gorm.io/gorm"
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
// Secondary Storage Config
// =======================

type SecondaryStorageConfig struct {
	Type            SecondaryStorageType
	MemoryOptions   *SecondaryStorageMemoryOptions
	DatabaseOptions *SecondaryStorageDatabaseOptions
	Storage         SecondaryStorage
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
	CookieName string
	ExpiresIn  time.Duration
	UpdateAge  time.Duration
}

// =======================
// CSRF Config
// =======================

type CSRFConfig struct {
	Enabled    bool
	CookieName string
	HeaderName string
	ExpiresIn  time.Duration
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
// Rate Limit Config
// =======================

type RateLimitCustomRule struct {
	Disabled bool
	Window   time.Duration
	Max      int
}

type RateLimitCustomRuleFunc func(req *http.Request) RateLimitCustomRule

type IPConfig struct {
	Headers []string
}

type RateLimitConfig struct {
	Enabled     bool
	Window      time.Duration
	Max         int
	Algorithm   string
	Prefix      string
	CustomRules map[string]RateLimitCustomRuleFunc
	IP          IPConfig
}

// =======================
// Endpoint Hooks Config
// =======================

type EndpointHookContext struct {
	Path            string
	Method          string
	Body            map[string]any
	Headers         map[string][]string
	Query           map[string][]string
	Request         *http.Request
	User            *User
	ResponseStatus  int
	ResponseBody    []byte
	ResponseHeaders map[string][]string
	ResponseCookies []*http.Cookie
}

type EndpointHooksConfig struct {
	Before   func(ctx *EndpointHookContext) error
	Response func(ctx *EndpointHookContext) error
	After    func(ctx *EndpointHookContext) error
}

// =======================
// Database Hooks Config
// =======================

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

type DatabaseHooksConfig struct {
	Users         *UserDatabaseHooksConfig
	Accounts      *AccountDatabaseHooksConfig
	Sessions      *SessionDatabaseHooksConfig
	Verifications *VerificationDatabaseHooksConfig
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
// Event Bus Config
// =======================

type EventBusConfig struct {
	Enabled bool
	Prefix  string
	PubSub  PubSub
}

type PluginsConfig struct {
	Plugins []Plugin
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
	DB                *gorm.DB
	Database          DatabaseConfig
	SecondaryStorage  SecondaryStorageConfig
	EmailPassword     EmailPasswordConfig
	EmailVerification EmailVerificationConfig
	User              UserConfig
	Session           SessionConfig
	CSRF              CSRFConfig
	SocialProviders   SocialProvidersConfig
	TrustedOrigins    TrustedOriginsConfig
	RateLimit         RateLimitConfig
	EndpointHooks     EndpointHooksConfig
	DatabaseHooks     DatabaseHooksConfig
	EventHooks        EventHooksConfig
	EventBus          EventBusConfig
	Plugins           PluginsConfig
}

type ConfigOption func(*Config)
