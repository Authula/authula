package types

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
