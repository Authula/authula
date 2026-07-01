package types

import "time"

type ApiKeyPluginConfig struct {
	AllowOrgKeys    bool          `json:"allow_org_keys" toml:"allow_org_keys"`
	DefaultPrefix   string        `json:"default_prefix" toml:"default_prefix"`
	Header          string        `json:"header" toml:"header"`
	AutoCleanup     bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
}

func (c *ApiKeyPluginConfig) ApplyDefaults() {
	if c.Header == "" {
		c.Header = "X-AUTHULA-API-KEY"
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = time.Minute
	}
}
