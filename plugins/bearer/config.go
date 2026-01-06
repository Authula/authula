package bearer

// BearerPluginConfig defines configuration options for the bearer plugin.
type BearerPluginConfig struct {
	Enabled    bool   `json:"enabled" toml:"enabled"`
	HeaderName string `json:"header_name" toml:"header_name"`
}

func (config *BearerPluginConfig) ApplyDefaults() {
	if config.HeaderName == "" {
		config.HeaderName = "Authorization"
	}
}
