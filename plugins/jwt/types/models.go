package types

import (
	"time"

	"github.com/uptrace/bun"
)

type JWKS struct {
	bun.BaseModel `bun:"table:jwks"`

	ID         string     `json:"id" bun:"column:id,pk"`
	PublicKey  string     `json:"public_key" bun:"column:public_key"`
	PrivateKey string     `json:"private_key" bun:"column:private_key"`
	ExpiresAt  *time.Time `json:"expires_at" bun:"column:expires_at"`
	CreatedAt  time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
}

type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	ID               string     `json:"id" bun:"column:id,pk"`
	SessionID        string     `json:"session_id" bun:"column:session_id"`
	TokenHash        string     `json:"token_hash" bun:"column:token_hash"`
	ExpiresAt        time.Time  `json:"expires_at" bun:"column:expires_at"`
	IsRevoked        bool       `json:"is_revoked" bun:"column:is_revoked"`
	RevokedAt        *time.Time `json:"revoked_at" bun:"column:revoked_at"`
	LastReuseAttempt *time.Time `json:"last_reuse_attempt" bun:"column:last_reuse_attempt"`
	CreatedAt        time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
}
