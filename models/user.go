package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID            string    `json:"id" bun:",pk"`
	Name          string    `json:"name" bun:",notnull"`
	Email         string    `json:"email" bun:",unique,notnull"`
	EmailVerified bool      `json:"email_verified" bun:",notnull,default:false"`
	Image         *string   `json:"image"`
	CreatedAt     time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (s *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
