package domain

import "time"

type Account struct {
	ID                    string       `json:"id" gorm:"primaryKey"`
	UserID                string       `json:"user_id" gorm:"index"`
	AccountID             string       `json:"account_id"`
	ProviderID            ProviderType `json:"provider_id"`
	AccessToken           *string      `json:"access_token,omitempty"`
	RefreshToken          *string      `json:"refresh_token,omitempty"`
	IDToken               *string      `json:"id_token,omitempty"`
	AccessTokenExpiresAt  *time.Time   `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time   `json:"refresh_token_expires_at,omitempty"`
	Scope                 *string      `json:"scope,omitempty"`
	Password              *string      `json:"password,omitempty"` // for email/password auth
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
	User                  User         `gorm:"foreignKey:UserID"`
}
