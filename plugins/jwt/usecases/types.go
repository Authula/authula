package usecases

import (
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

type JWKSResult struct {
	KeySet jwk.Set
}
