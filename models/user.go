package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID            string          `json:"id" bun:"id,pk"`
	Name          string          `json:"name" bun:"name,notnull"`
	Email         string          `json:"email" bun:"email,unique,notnull"`
	EmailVerified bool            `json:"email_verified" bun:"email_verified,notnull,default:false"`
	Image         *string         `json:"image" bun:"image"`
	Metadata      json.RawMessage `json:"metadata" bun:"metadata,type:json,notnull"`
	CreatedAt     time.Time       `json:"created_at" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time       `json:"updated_at" bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
