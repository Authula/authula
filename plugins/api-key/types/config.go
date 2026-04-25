package types

type ApiKeyPluginConfig struct {
	AllowOrgKeys  bool   `json:"allow_org_keys" toml:"allow_org_keys"`
	DefaultPrefix string `json:"default_prefix" toml:"default_prefix"`
	ApiKeyHeader  string `json:"api_key_header" toml:"api_key_header"`
}

func (c *ApiKeyPluginConfig) ApplyDefaults() {
	if c.ApiKeyHeader == "" {
		c.ApiKeyHeader = "X-API-KEY"
	}

}
