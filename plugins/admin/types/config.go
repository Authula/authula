package types

import "time"

type AdminPluginConfig struct {
	Enabled                   bool          `json:"enabled" toml:"enabled"`
	ImpersonationMaxExpiresIn time.Duration `json:"impersonation_max_expires_in" toml:"impersonation_max_expires_in"`
}

func (config *AdminPluginConfig) ApplyDefaults() {
	if config.ImpersonationMaxExpiresIn == 0 {
		config.ImpersonationMaxExpiresIn = 15 * time.Minute
	}
}
