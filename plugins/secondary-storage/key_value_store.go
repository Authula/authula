package secondarystorage

import (
	"time"

	"github.com/uptrace/bun"
)

// KeyValueStore represents the persistent key-value store table in the database.
// This is a domain model used for secondary storage operations.
type KeyValueStore struct {
	bun.BaseModel `bun:"table:key_value_store"`

	Key       string     `json:"key" bun:"column:key,pk"`
	Value     string     `json:"value" bun:"column:value"`
	ExpiresAt *time.Time `json:"expires_at" bun:"column:expires_at"`
	CreatedAt time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}
