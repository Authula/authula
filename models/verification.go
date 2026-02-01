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
