package services

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type UserService interface {
	Create(ctx context.Context, name string, email string, emailVerified bool, image *string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	UpdateFields(ctx context.Context, id string, fields map[string]any) error
}

type AccountService interface {
	Create(ctx context.Context, userID string, accountID string, providerID string, password *string) (*models.Account, error)
	CreateOAuth2(ctx context.Context, userID string, providerAccountID string, provider string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error)
	GetByUserID(ctx context.Context, userID string) (*models.Account, error)
	GetByUserIDAndProvider(ctx context.Context, userID string, provider string) (*models.Account, error)
	GetByProviderAndAccountID(ctx context.Context, provider string, accountID string) (*models.Account, error)
	Update(ctx context.Context, account *models.Account) (*models.Account, error)
	UpdateFields(ctx context.Context, userID string, fields map[string]any) error
}

type SessionService interface {
	GetByID(ctx context.Context, id string) (*models.Session, error)
	Create(ctx context.Context, userID string, hashedToken string, ipAddress *string, userAgent *string, maxAge time.Duration) (*models.Session, error)
	GetByUserID(ctx context.Context, userID string) (*models.Session, error)
	GetByToken(ctx context.Context, hashedToken string) (*models.Session, error)
	Update(ctx context.Context, session *models.Session) (*models.Session, error)
	Delete(ctx context.Context, ID string) error
	DeleteAllByUserID(ctx context.Context, userID string) error
}

type VerificationService interface {
	Create(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error)
	GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error
	IsExpired(verif *models.Verification) bool
}

type TokenService interface {
	Generate() (string, error)
	Hash(token string) string
	Encrypt(token string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type CoreServices struct {
	UserService         UserService
	AccountService      AccountService
	SessionService      SessionService
	VerificationService VerificationService
	TokenService        TokenService
}
