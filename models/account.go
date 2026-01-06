package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"table:accounts,alias:a"`

	ID                    string     `json:"id" bun:",pk"`
	UserID                string     `json:"user_id" bun:",notnull"`
	AccountID             string     `json:"account_id" bun:",unique:idx_accounts_provider_account,notnull"`
	ProviderID            string     `json:"provider_id" bun:",unique:idx_accounts_provider_account,notnull"`
	AccessToken           *string    `json:"access_token"`
	RefreshToken          *string    `json:"refresh_token"`
	IDToken               *string    `json:"id_token"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at"`
	Scope                 *string    `json:"scope"`
	Password              *string    `json:"password"` // for email/password auth
	CreatedAt             time.Time  `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt             time.Time  `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	User User `json:"-" bun:"rel:belongs-to,join:user_id=id"`
}

var _ bun.BeforeAppendModelHook = (*Account)(nil)

func (s *Account) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		s.UpdatedAt = time.Now()
	}
	return nil
}
