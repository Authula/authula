package session

type SessionPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`
}

func (config *SessionPluginConfig) ApplyDefaults() {

}
