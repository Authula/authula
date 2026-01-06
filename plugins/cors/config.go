package cors

import "time"

type CORSPluginConfig struct {
	Enabled          bool          `json:"enabled" toml:"enabled"`
	AllowedOrigins   []string      `json:"allowed_origins" toml:"allowed_origins"`
	AllowedMethods   []string      `json:"allowed_methods" toml:"allowed_methods"`
	AllowedHeaders   []string      `json:"allowed_headers" toml:"allowed_headers"`
	ExposedHeaders   []string      `json:"exposed_headers" toml:"exposed_headers"`
	AllowCredentials bool          `json:"allow_credentials" toml:"allow_credentials"`
	MaxAge           time.Duration `json:"max_age" toml:"max_age"`
}

func (config *CORSPluginConfig) ApplyDefaults() {
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Content-Type", "Authorization", "Cookie", "Set-Cookie"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}
}
