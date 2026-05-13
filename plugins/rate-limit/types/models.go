package types

import (
	"time"

	"github.com/uptrace/bun"
)

type RateLimit struct {
	bun.BaseModel `bun:"table:rate_limits"`

	Key       string    `json:"key" bun:"column:key,pk"`
	Count     int       `json:"count" bun:"column:count"`
	ExpiresAt time.Time `json:"expires_at" bun:"column:expires_at"`
}

type RateLimitRuleRecord struct {
	bun.BaseModel `bun:"table:rate_limit_rules"`

	Key           string `json:"key" bun:"column:key,pk"`
	WindowSeconds int    `json:"window_seconds" bun:"column:window_seconds"`
	MaxRequests   int    `json:"max_requests" bun:"column:max_requests"`
}
