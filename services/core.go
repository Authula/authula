package services

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
)

type UserService interface {
	GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error)
	Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata map[string]any) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	UpdateFields(ctx context.Context, id string, fields map[string]any) error
	Delete(ctx context.Context, id string) error
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
	DeleteAllExpired(ctx context.Context) error
	GetDistinctUserIDs(ctx context.Context) ([]string, error)
	DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error
}

type VerificationService interface {
	Create(ctx context.Context, userID string, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error)
	GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error
	IsExpired(verif *models.Verification) bool
	DeleteExpired(ctx context.Context) error
}

type TokenService interface {
	Generate() (string, error)
	Hash(token string) string
	Encrypt(token string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type PasswordService interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) bool
}

type CoreServices struct {
	UserService         UserService
	AccountService      AccountService
	SessionService      SessionService
	VerificationService VerificationService
	TokenService        TokenService
	PasswordService     PasswordService
}

// RateLimitKeyRule holds the per-key rate-limit configuration seeded at key creation.
type RateLimitKeyRule struct {
	Window      time.Duration
	MaxRequests int
}

type RateLimiterService interface {
	// GetValue returns a key's value
	GetValue(ctx context.Context, key string) (any, error)
	// CheckAndIncrement checks the counter for an arbitrary key and increments it.
	// Used for general IP-based rate limiting.
	CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (allowed bool, count int, resetAt time.Time, err error)
	// SetRule stores a per-key rate-limit rule without consuming quota.
	// Used when an API key with rate limiting is created.
	SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error
	// GetRule retrieves a previously stored per-key rule.
	// Returns nil, nil when no rule exists for the given key.
	GetRule(ctx context.Context, key string) (*RateLimitKeyRule, error)
	// DeleteRule removes the stored rule for a key.
	// Used when an API key is deleted.
	DeleteRule(ctx context.Context, key string) error
}

type OrganizationService interface {
	ExistsByID(ctx context.Context, organizationID string) (bool, error)
}
