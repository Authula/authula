package types

import (
	"errors"
	"time"

	"github.com/Authula/authula/plugins/jwt/constants"
)

type JWTAlgorithm string

const (
	JWTAlgEdDSA  JWTAlgorithm = "eddsa"
	JWTAlgRS256  JWTAlgorithm = "rs256"
	JWTAlgPS256  JWTAlgorithm = "ps256"
	JWTAlgES256  JWTAlgorithm = "es256"
	JWTAlgES512  JWTAlgorithm = "es512"
	JWTAlgECDHES JWTAlgorithm = "ecdh-es"
)

func (a JWTAlgorithm) String() string {
	return string(a)
}

// ValidateAlgorithm enforces that the algorithm can be used for JWT signing
func ValidateAlgorithm(alg JWTAlgorithm) error {
	switch alg {
	case JWTAlgEdDSA, JWTAlgRS256, JWTAlgPS256, JWTAlgES256, JWTAlgES512:
		return nil
	case JWTAlgECDHES:
		return errors.New("ECDH-ES cannot be used for JWT signing")
	default:
		return errors.New("unsupported JWT algorithm")
	}
}

type JWTTokenType string

const (
	JWTTokenTypeAccess  JWTTokenType = "access_token"
	JWTTokenTypeRefresh JWTTokenType = "refresh_token"
)

func (t JWTTokenType) String() string {
	return string(t)
}

// ParseAlgorithm parses a string into an Algorithm, accepting only canonical names (case-insensitive input)
func ParseAlgorithm(s string) (JWTAlgorithm, error) {
	switch s {
	case "eddsa":
		return JWTAlgEdDSA, nil
	case "rs256":
		return JWTAlgRS256, nil
	case "ps256":
		return JWTAlgPS256, nil
	case "es256":
		return JWTAlgES256, nil
	case "es512":
		return JWTAlgES512, nil
	case "ecdh-es":
		return JWTAlgECDHES, nil
	default:
		return "", errors.New("unsupported jwt algorithm")
	}
}

// Claims represents standard JWT claims
type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"sid"`
	Type      string `json:"type"` // "access_token" or "refresh_token"
	Sub       string `json:"sub"`
	Iss       string `json:"iss"`
	Aud       string `json:"aud"`
	Exp       int64  `json:"exp"`
	Iat       int64  `json:"iat"`
	Nbf       int64  `json:"nbf,omitempty"`
	Jti       string `json:"jti"`
}

// TokenPair holds both access and refresh tokens
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
