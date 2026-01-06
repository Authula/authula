package session

import "time"

type SessionPluginConfig struct {
	Enabled    bool          `json:"enabled" toml:"enabled"`
	CookieName string        `json:"cookie_name" toml:"cookie_name"`
	CookiePath string        `json:"cookie_path" toml:"cookie_path"`
	MaxAge     time.Duration `json:"max_age" toml:"max_age"`
	Secure     bool          `json:"secure" toml:"secure"`
	HttpOnly   bool          `json:"http_only" toml:"http_only"`
	SameSite   string        `json:"same_site" toml:"same_site"`
}

func (config *SessionPluginConfig) ApplyDefaults() {
	if config.CookieName == "" {
		config.CookieName = "gobetterauth.session_token"
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}
	if config.SameSite == "" {
		config.SameSite = "lax"
	}
}
