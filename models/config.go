package models

import (
	"net/http"
	"time"

	"gorm.io/gorm"
)

// =======================
// Mode
// =======================

type Mode string

const (
	ModeLibrary    Mode = "library"    // Library mode: embedded in another application
	ModeStandalone Mode = "standalone" // Standalone mode: standalone server with database-backed configuration
)

// =======================
// Logger Config
// =======================

type LoggerConfig struct {
	Level  string `json:"level" toml:"level"`
	Logger Logger `json:"-" toml:"-"`
}

// =======================
// Database Config
// =======================

type DatabaseConfig struct {
	Provider        string        `json:"provider" toml:"provider"`
	URL             string        `json:"url" toml:"url"`
	MaxOpenConns    int           `json:"max_open_conns" toml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" toml:"conn_max_lifetime"`
}

// =======================
// Secondary Storage Config
// =======================

type SecondaryStorageMemoryOptions struct {
	// CleanupInterval controls how often expired entries are cleaned up.
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

type SecondaryStorageDatabaseOptions struct {
	// CleanupInterval controls how often expired entries are cleaned up.
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

type SecondaryStorageConfig struct {
	Type            SecondaryStorageType            `json:"type" toml:"type"`
	MemoryOptions   SecondaryStorageMemoryOptions   `json:"memory_options" toml:"memory_options"`
	DatabaseOptions SecondaryStorageDatabaseOptions `json:"database_options" toml:"database_options"`
	Storage         SecondaryStorage                `json:"-" toml:"-"`
}

// =======================
// Email Config
// =======================

type EmailConfig struct {
	Provider string `json:"provider" toml:"provider"`
	SMTPHost string `json:"smtp_host" toml:"smtp_host"`
	SMTPPort int    `json:"smtp_port" toml:"smtp_port"`
	SMTPUser string `json:"smtp_user" toml:"smtp_user"`
	SMTPPass string `json:"smtp_pass" toml:"smtp_pass"`
	From     string `json:"from" toml:"from"`
}

// =======================
// Email Password Config
// =======================

// Library mode only
type PasswordConfig struct {
	Hash   func(password string) (string, error)      `json:"-" toml:"-"`
	Verify func(hashedPassword, password string) bool `json:"-" toml:"-"`
}

type EmailPasswordConfig struct {
	Enabled                  bool          `json:"enabled" toml:"enabled"`
	MinPasswordLength        int           `json:"min_password_length" toml:"min_password_length"`
	MaxPasswordLength        int           `json:"max_password_length" toml:"max_password_length"`
	DisableSignUp            bool          `json:"disable_sign_up" toml:"disable_sign_up"`
	RequireEmailVerification bool          `json:"require_email_verification" toml:"require_email_verification"`
	AutoSignIn               bool          `json:"auto_sign_in" toml:"auto_sign_in"`
	ResetTokenExpiry         time.Duration `json:"reset_token_expiry" toml:"reset_token_expiry"`
	// Library mode only
	Password               PasswordConfig                                  `json:"-" toml:"-"`
	SendResetPasswordEmail func(user User, url string, token string) error `json:"-" toml:"-"`
}

// =======================
// Email Verification Config
// =======================

type EmailVerificationConfig struct {
	AutoSignIn   bool          `json:"auto_sign_in" toml:"auto_sign_in"`
	SendOnSignUp bool          `json:"send_on_sign_up" toml:"send_on_sign_up"`
	SendOnSignIn bool          `json:"send_on_sign_in" toml:"send_on_sign_in"`
	ExpiresIn    time.Duration `json:"expires_in" toml:"expires_in"`
	// Library mode only
	SendVerificationEmail func(user User, url string, token string) error `json:"-" toml:"-"`
}

// =======================
// User Config
// =======================

type ChangeEmailConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`
	// Library mode only
	SendEmailChangeVerificationEmail func(user User, newEmail string, url string, token string) error `json:"-" toml:"-"`
}

type UserConfig struct {
	ChangeEmail ChangeEmailConfig `json:"change_email" toml:"change_email"`
}

// =======================
// Session Config
// =======================

type SessionConfig struct {
	CookieName string        `json:"cookie_name" toml:"cookie_name"`
	ExpiresIn  time.Duration `json:"expires_in" toml:"expires_in"`
	UpdateAge  time.Duration `json:"update_age" toml:"update_age"`
}

// =======================
// CSRF Config
// =======================

type CSRFConfig struct {
	Enabled    bool          `json:"enabled" toml:"enabled"`
	CookieName string        `json:"cookie_name" toml:"cookie_name"`
	HeaderName string        `json:"header_name" toml:"header_name"`
	ExpiresIn  time.Duration `json:"expires_in" toml:"expires_in"`
}

// =======================
// Social Providers Config
// =======================

type OAuth2ProviderConfig struct {
	Enabled      bool     `json:"enabled" toml:"enabled"`
	ClientID     string   `json:"client_id" toml:"client_id"`
	ClientSecret string   `json:"client_secret" toml:"client_secret"`
	RedirectURL  string   `json:"redirect_url" toml:"redirect_url"`
	Scopes       []string `json:"scopes" toml:"scopes"`
	// For generic providers or overriding defaults
	AuthURL     string `json:"auth_url" toml:"auth_url"`
	TokenURL    string `json:"token_url" toml:"token_url"`
	UserInfoURL string `json:"user_info_url" toml:"user_info_url"`
}

type SocialProvidersConfig map[string]OAuth2ProviderConfig

// =======================
// Trusted Origins Config
// =======================

type TrustedOriginsConfig struct {
	Origins []string `json:"origins" toml:"origins"`
}

// =======================
// Rate Limit Config
// =======================

type RateLimitCustomRule struct {
	Disabled bool          `json:"disabled" toml:"disabled"`
	Window   time.Duration `json:"window" toml:"window"`
	Max      int           `json:"max" toml:"max"`
}

type IPConfig struct {
	Headers []string `json:"headers" toml:"headers"`
}

type RateLimitConfig struct {
	Enabled     bool                           `json:"enabled" toml:"enabled"`
	Window      time.Duration                  `json:"window" toml:"window"`
	Max         int                            `json:"max" toml:"max"`
	Algorithm   string                         `json:"algorithm" toml:"algorithm"`
	Prefix      string                         `json:"prefix" toml:"prefix"`
	CustomRules map[string]RateLimitCustomRule `json:"custom_rules" toml:"custom_rules"`
	IP          IPConfig                       `json:"ip" toml:"ip"`
}

// =======================
// Endpoint Hooks Config (Library mode only)
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
	ResponseHeaders map[string][]string
	ResponseBody    []byte
	ResponseCookies []*http.Cookie
	Redirect        func(url string, status int)
	Handled         bool
}

type EndpointHooksConfig struct {
	Before   func(ctx *EndpointHookContext) error
	Response func(ctx *EndpointHookContext) error
	After    func(ctx *EndpointHookContext)
}

// =======================
// Database Hooks Config (Library mode only)
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
// Event Hooks Config (Library mode only)
// =======================

type EventHooksConfig struct {
	OnUserSignedUp    func(user User)
	OnUserLoggedIn    func(user User)
	OnEmailVerified   func(user User)
	OnPasswordChanged func(user User)
	OnEmailChanged    func(user User)
}

// =======================
// Webhook Config
// =======================

type WebhookConfig struct {
	URL            string            `json:"url" toml:"url"`
	Headers        map[string]string `json:"headers" toml:"headers"`
	TimeoutSeconds time.Duration     `json:"timeout_seconds" toml:"timeout_seconds"`
}

type WebhooksConfig struct {
	OnUserSignedUp    *WebhookConfig `json:"on_user_signed_up" toml:"on_user_signed_up"`
	OnUserLoggedIn    *WebhookConfig `json:"on_user_logged_in" toml:"on_user_logged_in"`
	OnEmailVerified   *WebhookConfig `json:"on_email_verified" toml:"on_email_verified"`
	OnPasswordChanged *WebhookConfig `json:"on_password_changed" toml:"on_password_changed"`
	OnEmailChanged    *WebhookConfig `json:"on_email_changed" toml:"on_email_changed"`
}

// =======================
// Event Bus Config
// =======================

type EventBusConfig struct {
	Enabled               bool   `json:"enabled" toml:"enabled"`
	Prefix                string `json:"prefix" toml:"prefix"`
	MaxConcurrentHandlers int    `json:"max_concurrent_handlers" toml:"max_concurrent_handlers"`
	PubSubType            string `json:"pubsub_type" toml:"pubsub_type"`
	PubSub                PubSub `json:"-" toml:"-"`
}

// Library mode only
type PluginsConfig struct {
	Plugins []Plugin
}

// =======================
// Main Config Structure
// =======================

// Config holds all configurable options for the GoBetterAuth library.
type Config struct {
	Mode              Mode                    `json:"-" toml:"-"`
	AppName           string                  `json:"app_name" toml:"app_name"`
	BaseURL           string                  `json:"base_url" toml:"base_url"`
	BasePath          string                  `json:"base_path" toml:"base_path"`
	Secret            string                  `json:"secret" toml:"secret"`
	Logger            LoggerConfig            `json:"logger" toml:"logger"`
	DB                *gorm.DB                `json:"-" toml:"-"`
	Database          DatabaseConfig          `json:"database" toml:"database"`
	Email             EmailConfig             `json:"email" toml:"email"`
	SecondaryStorage  SecondaryStorageConfig  `json:"secondary_storage" toml:"secondary_storage"`
	EmailPassword     EmailPasswordConfig     `json:"email_password" toml:"email_password"`
	EmailVerification EmailVerificationConfig `json:"email_verification" toml:"email_verification"`
	User              UserConfig              `json:"user" toml:"user"`
	Session           SessionConfig           `json:"session" toml:"session"`
	CSRF              CSRFConfig              `json:"csrf" toml:"csrf"`
	SocialProviders   SocialProvidersConfig   `json:"social_providers" toml:"social_providers"`
	TrustedOrigins    TrustedOriginsConfig    `json:"trusted_origins" toml:"trusted_origins"`
	RateLimit         RateLimitConfig         `json:"rate_limit" toml:"rate_limit"`
	EndpointHooks     EndpointHooksConfig     `json:"-" toml:"-"`
	DatabaseHooks     DatabaseHooksConfig     `json:"-" toml:"-"`
	EventHooks        EventHooksConfig        `json:"-" toml:"-"`
	Webhooks          WebhooksConfig          `json:"webhooks" toml:"webhooks"`
	EventBus          EventBusConfig          `json:"event_bus" toml:"event_bus"`
	Plugins           PluginsConfig           `json:"-" toml:"-"`
}

type ConfigOption func(*Config)
