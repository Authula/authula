package secondarystorage

import (
	"time"

	"github.com/uptrace/bun"
)

// KeyValueStore represents the persistent key-value store table in the database.
// This is a domain model used for secondary storage operations.
type KeyValueStore struct {
	bun.BaseModel `bun:"table:key_value_store"`

	Key       string     `json:"key" bun:",pk,type:varchar(255)"`
	Value     string     `json:"value"`
	ExpiresAt *time.Time `json:"expires_at" bun:"expires_at,nullzero"`
	CreatedAt time.Time  `json:"created_at" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time  `json:"updated_at" bun:"updated_at,notnull,default:current_timestamp"`
}
