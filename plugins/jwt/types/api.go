package types

import (
	"time"

	"github.com/Authula/authula/plugins/jwt/constants"
)

type JWTTokenType string

const (
	JWTTokenTypeAccess  JWTTokenType = "access_token"
	JWTTokenTypeRefresh JWTTokenType = "refresh_token"
)

func (t JWTTokenType) String() string {
	return string(t)
}

type TokenClaims struct {
	Subject        string   `json:"sub"`
	UserID         string   `json:"user_id,omitempty"`
	SessionID      string   `json:"session_id,omitempty"`
	TokenType      string   `json:"token_type"`
	ActorType      string   `json:"actor_type,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	JTI            string   `json:"jti"`
	IssuedAt       int64    `json:"iat"`
	Expiration     int64    `json:"exp"`
}

type TokenPair struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
	TokenType    string        `json:"token_type"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return constants.ErrRefreshTokenIsRequired
	}
	return nil
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
