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

	ID         string           `json:"id" bun:"id,pk"`
	UserID     *string          `json:"user_id" bun:"user_id,nullzero"`
	Identifier string           `json:"identifier" bun:"identifier,notnull"` // email or other identifier
	Token      string           `json:"token" bun:"token,unique,notnull"`
	Type       VerificationType `json:"type" bun:"type,notnull"`
	ExpiresAt  time.Time        `json:"expires_at" bun:"expires_at,notnull"`
	CreatedAt  time.Time        `json:"created_at" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt  time.Time        `json:"updated_at" bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	User *User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

var _ bun.BeforeAppendModelHook = (*Verification)(nil)

func (v *Verification) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		v.CreatedAt = time.Now()
		v.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		v.UpdatedAt = time.Now()
	}
	return nil
}
