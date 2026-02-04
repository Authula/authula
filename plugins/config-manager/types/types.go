package types

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type ConfigManagerPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`
}

// AuthSettings stores dynamic configuration for the auth system in the database.
// This is used primarily in database mode to persist the full runtime configuration.
type AuthSettings struct {
	bun.BaseModel `bun:"table:auth_settings"`

	// ConfigVersion is the version number for optimistic concurrency control
	ConfigVersion int64 `json:"config_version" bun:"column:config_version,pk"`
	// The unique key for the config block (e.g., "runtime_config" for the main config)
	Key string `json:"key" bun:"column:key"`
	// Value contains the JSON-encoded configuration data
	Value json.RawMessage `json:"value" bun:"column:value"`
	// CreatedAt is the timestamp when this setting was created
	CreatedAt time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	// UpdatedAt is the timestamp when this setting was last updated
	UpdatedAt time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}
