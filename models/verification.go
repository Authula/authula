package models

import (
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
	bun.BaseModel `bun:"table:verifications"`

	ID         string           `json:"id" bun:"column:id,pk"`
	UserID     *string          `json:"user_id" bun:"column:user_id"`
	Identifier string           `json:"identifier" bun:"column:identifier"` // email or other identifier
	Token      string           `json:"token" bun:"column:token"`
	Type       VerificationType `json:"type" bun:"column:type"`
	ExpiresAt  time.Time        `json:"expires_at" bun:"column:expires_at"`
	CreatedAt  time.Time        `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt  time.Time        `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`

	User *User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}
