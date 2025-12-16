package models

import "time"

type VerificationType string

const (
	TypeEmailVerification VerificationType = "email_verification"
	TypePasswordReset     VerificationType = "password_reset"
	TypeEmailChange       VerificationType = "email_change"
)

type Verification struct {
	ID         string           `json:"id" gorm:"primaryKey"`
	UserID     *string          `json:"user_id,omitempty"`
	Identifier string           `json:"identifier"` // email or other identifier
	Token      string           `json:"token"`
	Type       VerificationType `json:"type"`
	ExpiresAt  time.Time        `json:"expires_at"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}
