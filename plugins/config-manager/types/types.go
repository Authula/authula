package types

import (
	"context"
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
	bun.BaseModel `bun:"table:auth_settings,alias:as"`

	// ConfigVersion is the version number for optimistic concurrency control
	ConfigVersion int64 `json:"config_version" bun:",pk,autoincrement"`
	// The unique key for the config block (e.g., "runtime_config" for the main config)
	Key string `json:"key" bun:",unique,type:varchar(255)"`
	// Value contains the JSON-encoded configuration data
	Value json.RawMessage `json:"value" bun:"type:jsonb"`
	// CreatedAt is the timestamp when this setting was created
	CreatedAt time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	// UpdatedAt is the timestamp when this setting was last updated
	UpdatedAt time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`
}

// TableName specifies the table name for the AuthSettings model
func (AuthSettings) TableName() string {
	return "auth_settings"
}

var _ bun.BeforeAppendModelHook = (*AuthSettings)(nil)

func (s *AuthSettings) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
