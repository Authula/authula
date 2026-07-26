package models

import (
	"time"

	"github.com/swaggest/jsonschema-go"
	"github.com/uptrace/bun"
)

type VerificationType string

const (
	TypeEmailVerification      VerificationType = "email_verification"
	TypePasswordResetRequest   VerificationType = "password_reset_request"
	TypeEmailResetRequest      VerificationType = "email_reset_request"
	TypeMagicLinkSignInRequest VerificationType = "magic_link_sign_in_request"
	TypeMagicLinkExchangeCode  VerificationType = "magic_link_exchange_code"
	TypeTOTPPendingAuth        VerificationType = "totp_pending_auth"
)

func (vt VerificationType) String() string {
	return string(vt)
}

func (VerificationType) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithType(jsonschema.String.Type())
	schema.Enum = []any{
		string(TypeEmailVerification),
		string(TypePasswordResetRequest),
		string(TypeEmailResetRequest),
		string(TypeMagicLinkSignInRequest),
		string(TypeMagicLinkExchangeCode),
		string(TypeTOTPPendingAuth),
	}
	schema.WithDescription("The type of the verification")
	return nil
}

type Verification struct {
	bun.BaseModel `bun:"table:verifications"`

	ID         string           `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	UserID     *string          `json:"user_id" required:"true" nullable:"false" bun:"column:user_id"`
	Identifier string           `json:"identifier" required:"true" nullable:"false" bun:"column:identifier"` // email or other identifier
	Token      string           `json:"token" required:"true" nullable:"false" bun:"column:token"`
	Type       VerificationType `json:"type" required:"true" nullable:"false" bun:"column:type"`
	ExpiresAt  time.Time        `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	CreatedAt  time.Time        `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt  time.Time        `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`

	User *User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}
