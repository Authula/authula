package types

type ApiKeyPluginConfig struct {
	AllowOrgKeys  bool   `json:"allow_org_keys" toml:"allow_org_keys"`
	DefaultPrefix string `json:"default_prefix" toml:"default_prefix"`
	Header        string `json:"header" toml:"header"`
}

func (c *ApiKeyPluginConfig) ApplyDefaults() {
	if c.Header == "" {
		c.Header = "X-API-KEY"
	}
}
