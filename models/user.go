package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID            string          `json:"id" bun:"id,pk"`
	Name          string          `json:"name" bun:"name,notnull"`
	Email         string          `json:"email" bun:"email,unique,notnull"`
	EmailVerified bool            `json:"email_verified" bun:"email_verified,notnull,default:false"`
	Image         *string         `json:"image" bun:"image"`
	Metadata      json.RawMessage `json:"metadata" bun:"metadata,type:json,notnull"`
	CreatedAt     time.Time       `json:"created_at" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time       `json:"updated_at" bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	if len(u.Metadata) == 0 {
		u.Metadata = json.RawMessage("{}")
	}

	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}
