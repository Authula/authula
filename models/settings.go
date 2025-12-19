package models

import (
	"encoding/json"
	"time"
)

// AuthSettings stores dynamic configuration for the auth system in the database.
// This is used primarily in database mode to persist the full runtime configuration.
type AuthSettings struct {
	// The unique key for the config block (e.g., "runtime_config" for the main config)
	Key string `gorm:"primaryKey;type:varchar(255)" json:"key"`
	// Value contains the JSON-encoded configuration data
	Value json.RawMessage `gorm:"type:jsonb" json:"value"`
	// CreatedAt is the timestamp when this setting was created
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	// UpdatedAt is the timestamp when this setting was last updated
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for the AuthSettings model
func (AuthSettings) TableName() string {
	return "auth_settings"
}
