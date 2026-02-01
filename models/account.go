package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"table:accounts"`

	ID                    string     `json:"id" bun:"id,pk"`
	UserID                string     `json:"user_id" bun:"user_id,notnull"`
	AccountID             string     `json:"account_id" bun:"account_id,unique:idx_accounts_provider_account,notnull"`
	ProviderID            string     `json:"provider_id" bun:"provider_id,unique:idx_accounts_provider_account,notnull"`
	AccessToken           *string    `json:"access_token" bun:"access_token"`
	RefreshToken          *string    `json:"refresh_token" bun:"refresh_token"`
	IDToken               *string    `json:"id_token" bun:"id_token"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at" bun:"access_token_expires_at"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at" bun:"refresh_token_expires_at"`
	Scope                 *string    `json:"scope" bun:"scope"`
	Password              *string    `json:"password" bun:"password"` // for email/password auth
	CreatedAt             time.Time  `json:"created_at" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt             time.Time  `json:"updated_at" bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	User *User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}
