package types

type AccessControlPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`
	// TODO: Add a field to enable auto-cleanup of expired user roles and permissions
}

func (config *AccessControlPluginConfig) ApplyDefaults() {}
