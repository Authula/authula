package services

import (
	"context"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type UserService interface {
	CreateUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	UpdateUser(user *models.User) error
}

type AccountService interface {
	CreateAccount(account *models.Account) error
	GetAccountByUserID(userID string) (*models.Account, error)
	GetAccountByProviderAndAccountID(provider models.ProviderType, accountID string) (*models.Account, error)
	UpdateAccount(account *models.Account) error
}

type SessionService interface {
	CreateSession(userID string, token string) (*models.Session, error)
	GetSessionByUserID(userID string) (*models.Session, error)
	GetSessionByToken(token string) (*models.Session, error)
	DeleteSessionByID(ID string) error
}

type VerificationService interface {
	CreateVerification(verif *models.Verification) error
	GetVerificationByToken(token string) (*models.Verification, error)
	DeleteVerification(id string) error
	IsExpired(verif *models.Verification) bool
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
