package types

type AccessControlPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`
}

func (config *AccessControlPluginConfig) ApplyDefaults() {}
