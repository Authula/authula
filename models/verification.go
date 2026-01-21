package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type VerificationType string

const (
	TypeEmailVerification    VerificationType = "email_verification"
	TypePasswordResetRequest VerificationType = "password_reset_request"
	TypeEmailResetRequest    VerificationType = "email_reset_request"
)

func (vt VerificationType) String() string {
	return string(vt)
}

type Verification struct {
	bun.BaseModel `bun:"table:verifications,alias:v"`

	ID         string           `json:"id" bun:",pk"`
	UserID     *string          `json:"user_id" bun:",nullzero"`
	Identifier string           `json:"identifier" bun:",notnull"` // email or other identifier
	Token      string           `json:"token" bun:",unique,notnull"`
	Type       VerificationType `json:"type" bun:",notnull"`
	ExpiresAt  time.Time        `json:"expires_at" bun:",notnull"`
	CreatedAt  time.Time        `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt  time.Time        `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	User *User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

var _ bun.BeforeAppendModelHook = (*Verification)(nil)

func (s *Verification) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
