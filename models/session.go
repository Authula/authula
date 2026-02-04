package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	ID        string    `json:"id" bun:"column:id,pk"`
	UserID    string    `json:"user_id" bun:"column:user_id"`
	Token     string    `json:"token" bun:"column:token"`
	ExpiresAt time.Time `json:"expires_at" bun:"column:expires_at"`
	IPAddress *string   `json:"ip_address" bun:"column:ip_address"`
	UserAgent *string   `json:"user_agent" bun:"column:user_agent"`
	CreatedAt time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`

	User User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}
