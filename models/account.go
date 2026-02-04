package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"table:accounts"`

	ID                    string     `json:"id" bun:"column:id,pk"`
	UserID                string     `json:"user_id" bun:"column:user_id"`
	AccountID             string     `json:"account_id" bun:"column:account_id"`
	ProviderID            string     `json:"provider_id" bun:"column:provider_id"`
	AccessToken           *string    `json:"access_token" bun:"column:access_token"`
	RefreshToken          *string    `json:"refresh_token" bun:"column:refresh_token"`
	IDToken               *string    `json:"id_token" bun:"column:id_token"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at" bun:"column:access_token_expires_at"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at" bun:"column:refresh_token_expires_at"`
	Scope                 *string    `json:"scope" bun:"column:scope"`
	Password              *string    `json:"password" bun:"column:password"` // for email/password auth
	CreatedAt             time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt             time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`

	User User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}
