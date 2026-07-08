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
	RefreshToken string `json:"refresh_token" required:"true" nullable:"false"`
}

func (r *RefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return constants.ErrRefreshTokenIsRequired
	}
	return nil
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token" required:"true" nullable:"false"`
	RefreshToken string `json:"refresh_token" required:"true" nullable:"false"`
}

type JWK struct {
	Kty string `json:"kty" required:"true" nullable:"false"`
	Crv string `json:"crv,omitempty" nullable:"false"`
	Kid string `json:"kid,omitempty" nullable:"false"`
	X   string `json:"x,omitempty" nullable:"false"`
	Alg string `json:"alg,omitempty" nullable:"false"`
	Use string `json:"use,omitempty" nullable:"false"`
	N   string `json:"n,omitempty" nullable:"false"`
	E   string `json:"e,omitempty" nullable:"false"`
}

type WellKnownJWKSResponse struct {
	Keys []JWK `json:"keys" required:"true" nullable:"false"`
}
