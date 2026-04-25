package types

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type ApiKey struct {
	bun.BaseModel `bun:"table:api_keys"`

	ID                   string          `json:"id" bun:"column:id,pk"`
	KeyHash              string          `json:"key_hash" bun:"column:key_hash"`
	Name                 string          `json:"name" bun:"column:name"`
	OwnerType            string          `json:"owner_type" bun:"column:owner_type"`
	ReferenceID          string          `json:"reference_id" bun:"column:reference_id"`
	Start                string          `json:"start" bun:"column:start"`
	Prefix               *string         `json:"prefix" bun:"column:prefix"`
	Enabled              bool            `json:"enabled" bun:"column:enabled"`
	RateLimitEnabled     bool            `json:"rate_limit_enabled" bun:"column:rate_limit_enabled"`
	RateLimitTimeWindow  *int            `json:"rate_limit_time_window" bun:"column:rate_limit_time_window"`
	RateLimitMaxRequests *int            `json:"rate_limit_max_requests" bun:"column:rate_limit_max_requests"`
	RequestsRemaining    *int            `json:"requests_remaining" bun:"column:requests_remaining"`
	LastRequestedAt      *time.Time      `json:"last_requested_at" bun:"column:last_requested_at"`
	ExpiresAt            *time.Time      `json:"expires_at" bun:"column:expires_at"`
	Permissions          []string        `json:"permissions" bun:"column:permissions"`
	Metadata             json.RawMessage `json:"metadata" bun:"column:metadata"`
	CreatedAt            time.Time       `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt            time.Time       `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}
