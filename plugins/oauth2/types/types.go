package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"golang.org/x/oauth2"
)

// OAuth2PluginConfig represents the OAuth2 plugin configuration
type OAuth2PluginConfig struct {
	Enabled   bool                      `json:"enabled" toml:"enabled"`
	Providers map[string]ProviderConfig `json:"providers" toml:"providers"`
}

// ApplyDefaults applies default values to the config
func (c *OAuth2PluginConfig) ApplyDefaults() {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
}

// ProviderConfig represents configuration for an OAuth2 provider
type ProviderConfig struct {
	Enabled      bool     `json:"enabled" toml:"enabled"`
	ClientID     string   `json:"client_id" toml:"client_id"`
	ClientSecret string   `json:"client_secret" toml:"client_secret"`
	RedirectURL  string   `json:"redirect_url" toml:"redirect_url"`
	Scopes       []string `json:"scopes" toml:"scopes"`
	AuthURL      string   `json:"auth_url" toml:"auth_url"`
	TokenURL     string   `json:"token_url" toml:"token_url"`
	UserInfoURL  string   `json:"user_info_url" toml:"user_info_url"`
	UserIDField  string   `json:"user_id_field" toml:"user_id_field"`
	EmailField   string   `json:"email_field" toml:"email_field"`
	NameField    string   `json:"name_field" toml:"name_field"`
	PictureField string   `json:"picture_field" toml:"picture_field"`
}

// OAuth2Provider interface defines the contract for OAuth2 providers
type OAuth2Provider interface {
	Name() string
	GetConfig() *oauth2.Config
	GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	RequiresPKCE() bool
}

// UserInfo represents normalized user information from OAuth2 providers
type UserInfo struct {
	ProviderAccountID string          `json:"provider_account_id"`
	Email             string          `json:"email"`
	Name              string          `json:"name"`
	Picture           string          `json:"picture"`
	Raw               json.RawMessage `json:"raw"`
}

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ProviderID string
	RedirectTo string
}

// AuthorizeResponse represents an authorization response
type AuthorizeResponse struct {
	AuthURL string `json:"auth_url"`
}

// CallbackRequest represents an OAuth2 callback request
type CallbackRequest struct {
	ProviderID string
	Code       string
	State      string
	Error      string
}

// CallbackResult represents the result of OAuth2 callback
type CallbackResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

// CallbackResponse represents an OAuth2 callback response
type CallbackResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	Provider string
	UserID   string
}

// RefreshResponse represents a token refresh response
type RefreshResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// LinkAccountRequest represents an account linking request
type LinkAccountRequest struct {
	Provider   string
	UserID     string
	RedirectTo string
}

// LinkAccountResponse represents an account linking response
type LinkAccountResponse struct {
	User *models.User `json:"user,omitempty"`
}
