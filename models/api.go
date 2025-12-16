package models

import (
	"context"
	"net/http"
)

type UserService interface {
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
}

type AccountService interface {
	CreateAccount(account *Account) error
	GetAccountByUserID(userID string) (*Account, error)
	GetAccountByProviderAndAccountID(provider ProviderType, accountID string) (*Account, error)
	UpdateAccount(account *Account) error
}

type SessionService interface {
	CreateSession(userID string, token string) (*Session, error)
	GetSessionByUserID(userID string) (*Session, error)
	GetSessionByToken(token string) (*Session, error)
	DeleteSessionByID(ID string) error
}

type VerificationService interface {
	CreateVerification(verif *Verification) error
	GetVerificationByToken(token string) (*Verification, error)
	DeleteVerification(id string) error
	IsExpired(verif *Verification) bool
}

type TokenService interface {
	GenerateToken() (string, error)
	HashToken(token string) string
	GenerateEncryptedToken() (string, error)
	EncryptToken(token string) (string, error)
	DecryptToken(encryptedToken string) (string, error)
}

type RateLimitService interface {
	Allow(ctx context.Context, key string, req *http.Request) (bool, error)
	GetClientIP(req *http.Request) string
	BuildKey(key string) string
}

type Api struct {
	Users         UserService
	Accounts      AccountService
	Sessions      SessionService
	Verifications VerificationService
	Tokens        TokenService
	RateLimit     RateLimitService
	// TODO: KeyValueStore KeyValueStoreService
}
