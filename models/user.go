package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID            string         `json:"id" bun:"column:id,pk"`
	Name          string         `json:"name" bun:"column:name"`
	Email         string         `json:"email" bun:"column:email"`
	EmailVerified bool           `json:"email_verified" bun:"column:email_verified"`
	Image         *string        `json:"image" bun:"column:image"`
	Metadata      map[string]any `json:"metadata" bun:"column:metadata"`
	CreatedAt     time.Time      `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt     time.Time      `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}
