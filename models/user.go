package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID            string         `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	Name          string         `json:"name" required:"true" nullable:"false" bun:"column:name"`
	Email         string         `json:"email" required:"true" nullable:"false" bun:"column:email"`
	EmailVerified bool           `json:"email_verified" required:"true" nullable:"false" bun:"column:email_verified"`
	Image         *string        `json:"image" nullable:"true" bun:"column:image"`
	Metadata      map[string]any `json:"metadata" nullable:"true" bun:"column:metadata"`
	CreatedAt     time.Time      `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt     time.Time      `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}
