package csrf

import (
	"time"
)

type CSRFPluginConfig struct {
	Enabled                bool          `json:"enabled" toml:"enabled"`
	CookieName             string        `json:"cookie_name" toml:"cookie_name"`
	HeaderName             string        `json:"header_name" toml:"header_name"`
	MaxAge                 time.Duration `json:"max_age" toml:"max_age"`
	Secure                 bool          `json:"secure" toml:"secure"`
	SameSite               string        `json:"same_site" toml:"same_site"`
	EnableHeaderProtection bool          `json:"enable_header_protection" toml:"enable_header_protection"`
}

func (config *CSRFPluginConfig) ApplyDefaults() {
	if config.CookieName == "" {
		config.CookieName = "gobetterauth_csrf_token"
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-GOBETTERAUTH-CSRF-TOKEN"
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}
	if config.SameSite == "" {
		config.SameSite = "lax"
	}
}
