package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Session struct {
	bun.BaseModel `bun:"table:sessions,alias:s"`

	ID        string    `json:"id" bun:",pk"`
	UserID    string    `json:"user_id" bun:",notnull"`
	Token     string    `json:"token" bun:",unique,notnull"`
	ExpiresAt time.Time `json:"expires_at" bun:",notnull"`
	IPAddress *string   `json:"ip_address"`
	UserAgent *string   `json:"user_agent"`
	CreatedAt time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	User User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

var _ bun.BeforeAppendModelHook = (*Session)(nil)

func (s *Session) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
