package repositories

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/uptrace/bun"
)

// Transaction represents a database transaction interface
type Transaction interface{}

// Repository interfaces for data access - these will be implemented by plugins

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	UpdateFields(ctx context.Context, id string, fields map[string]any) error
	WithTx(tx bun.IDB) UserRepository
}

type AccountRepository interface {
	GetByID(ctx context.Context, id string) (*models.Account, error)
	GetByUserID(ctx context.Context, userID string) (*models.Account, error)
	GetByUserIDAndProvider(ctx context.Context, userID string, provider string) (*models.Account, error)
	GetByProviderAndAccountID(ctx context.Context, provider string, accountID string) (*models.Account, error)
	Create(ctx context.Context, account *models.Account) (*models.Account, error)
	Update(ctx context.Context, account *models.Account) (*models.Account, error)
	UpdateFields(ctx context.Context, userID string, fields map[string]any) error
	WithTx(tx bun.IDB) AccountRepository
}

type SessionRepository interface {
	GetByID(ctx context.Context, id string) (*models.Session, error)
	GetByToken(ctx context.Context, token string) (*models.Session, error)
	GetByUserID(ctx context.Context, userID string) (*models.Session, error)
	Create(ctx context.Context, session *models.Session) (*models.Session, error)
	Update(ctx context.Context, session *models.Session) (*models.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	WithTx(tx bun.IDB) SessionRepository
}

type VerificationRepository interface {
	GetByID(ctx context.Context, id string) (*models.Verification, error)
	GetByToken(ctx context.Context, token string) (*models.Verification, error)
	Create(ctx context.Context, verification *models.Verification) (*models.Verification, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error
	WithTx(tx bun.IDB) VerificationRepository
}

type TokenRepository interface {
	Generate() (string, error)
	Hash(token string) string
	Encrypt(token string) (string, error)
	Decrypt(encrypted string) (string, error)
}
