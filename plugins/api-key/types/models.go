package types

import (
	"time"

	"github.com/uptrace/bun"
)

type ApiKey struct {
	bun.BaseModel `bun:"table:api_keys"`

	ID               string         `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	KeyHash          string         `json:"key_hash" required:"true" nullable:"false" bun:"column:key_hash"`
	Name             string         `json:"name" required:"true" nullable:"false" bun:"column:name"`
	OwnerType        string         `json:"owner_type" required:"true" nullable:"false" bun:"column:owner_type"`
	OwnerID          string         `json:"owner_id" required:"true" nullable:"false" bun:"column:owner_id"`
	Start            string         `json:"start" required:"true" nullable:"false" bun:"column:start"`
	Last             string         `json:"last" required:"true" nullable:"false" bun:"column:last"`
	Prefix           *string        `json:"prefix" nullable:"true" bun:"column:prefix"`
	Enabled          bool           `json:"enabled" required:"true" nullable:"false" bun:"column:enabled"`
	RateLimitEnabled bool           `json:"rate_limit_enabled" required:"true" nullable:"false" bun:"column:rate_limit_enabled"`
	LastRequestedAt  *time.Time     `json:"last_requested_at" nullable:"true" bun:"column:last_requested_at"`
	ExpiresAt        *time.Time     `json:"expires_at" nullable:"true" bun:"column:expires_at"`
	Permissions      []string       `json:"permissions" nullable:"true" bun:"column:permissions"`
	Metadata         map[string]any `json:"metadata" nullable:"true" bun:"column:metadata"`
	CreatedAt        time.Time      `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt        time.Time      `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}
