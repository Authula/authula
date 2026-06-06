package tests

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/repositories"
	jwtservicesTypes "github.com/Authula/authula/plugins/jwt/types"
)

type InMemoryStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{data: make(map[string]string)}
}

func (s *InMemoryStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (s *InMemoryStorage) Set(_ context.Context, key string, value any, _ *time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value.(string)
	return nil
}

func (s *InMemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *InMemoryStorage) Incr(_ context.Context, _ string, _ *time.Duration) (int, error) {
	return 0, nil
}

func (s *InMemoryStorage) TTL(_ context.Context, _ string) (*time.Duration, error) {
	return nil, nil
}

func (s *InMemoryStorage) Scan(_ context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []string
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *InMemoryStorage) Close() error {
	return nil
}

type NopTokenService struct{}

func (NopTokenService) Generate() (string, error)                { return "", nil }
func (NopTokenService) Hash(token string) string                 { return token }
func (NopTokenService) Encrypt(token string) (string, error)     { return token, nil }
func (NopTokenService) Decrypt(encrypted string) (string, error) { return encrypted, nil }

var _ repositories.JWKSRepository = (*MockJWKSRepository)(nil)

type MockTokenService struct{ mock.Mock }

func (m *MockTokenService) GenerateUserToken(ctx context.Context, userID string, sessionID string) (*jwtservicesTypes.TokenPair, error) {
	args := m.Called(ctx, userID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.TokenPair), args.Error(1)
}

func (m *MockTokenService) GenerateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string) (*jwtservicesTypes.TokenPair, error) {
	args := m.Called(ctx, clientID, organizationID, scopes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.TokenPair), args.Error(1)
}

func (m *MockTokenService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Actor), args.Error(1)
}

type MockKeyService struct{ mock.Mock }

func (m *MockKeyService) GenerateKeysIfMissing(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockKeyService) GetActiveKey(ctx context.Context) (*jwtservicesTypes.JWKS, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.JWKS), args.Error(1)
}

func (m *MockKeyService) IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool {
	args := m.Called(ctx, rotationInterval)
	return args.Bool(0)
}

func (m *MockKeyService) RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error) {
	args := m.Called(ctx, rotationInterval, gracePeriod, invalidateCacheFunc)
	return args.Bool(0), args.Error(1)
}

type MockCacheService struct{ mock.Mock }

func (m *MockCacheService) GetCachedJWKS(ctx context.Context) (jwk.Set, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(jwk.Set), args.Error(1)
}

func (m *MockCacheService) FetchJWKSFromDatabase(ctx context.Context) (jwk.Set, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(jwk.Set), args.Error(1)
}

func (m *MockCacheService) CacheJWKS(ctx context.Context, set jwk.Set) error {
	args := m.Called(ctx, set)
	return args.Error(0)
}

func (m *MockCacheService) InvalidateCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheService) GetJWKSWithFallback(ctx context.Context) (jwk.Set, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(jwk.Set), args.Error(1)
}

type MockBlacklistService struct{ mock.Mock }

func (m *MockBlacklistService) BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error {
	args := m.Called(ctx, jti, expiresAt)
	return args.Error(0)
}

func (m *MockBlacklistService) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	args := m.Called(ctx, jti)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlacklistService) BlacklistAllSessionTokens(ctx context.Context, sessionID string, expiresAt time.Time) error {
	args := m.Called(ctx, sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockBlacklistService) CleanupExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockRefreshTokenRepository struct{ mock.Mock }

func (m *MockRefreshTokenRepository) StoreRefreshToken(ctx context.Context, record *jwtservicesTypes.RefreshToken) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*jwtservicesTypes.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeAllSessionTokens(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) SetLastReuseAttempt(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockJWKSRepository struct{ mock.Mock }

func (m *MockJWKSRepository) GetJWKSKeys(ctx context.Context) ([]*jwtservicesTypes.JWKS, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jwtservicesTypes.JWKS), args.Error(1)
}

func (m *MockJWKSRepository) GetJWKSKeyByID(ctx context.Context, id string) (*jwtservicesTypes.JWKS, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.JWKS), args.Error(1)
}

func (m *MockJWKSRepository) StoreJWKSKey(ctx context.Context, key *jwtservicesTypes.JWKS) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockJWKSRepository) UpdateJWKSKey(ctx context.Context, key *jwtservicesTypes.JWKS) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockJWKSRepository) MarkKeyExpired(ctx context.Context, id string, expiresAt time.Time) error {
	args := m.Called(ctx, id, expiresAt)
	return args.Error(0)
}

func (m *MockJWKSRepository) PurgeExpiredKeys(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockRefreshTokenService struct{ mock.Mock }

func (m *MockRefreshTokenService) RefreshTokens(ctx context.Context, refreshToken string) (*jwtservicesTypes.RefreshTokenResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservicesTypes.RefreshTokenResponse), args.Error(1)
}

func (m *MockRefreshTokenService) StoreInitialRefreshToken(ctx context.Context, refreshToken string, sessionID string, expiresAt time.Time) error {
	args := m.Called(ctx, refreshToken, sessionID, expiresAt)
	return args.Error(0)
}

type MockJWTService struct{ mock.Mock }

func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Actor), args.Error(1)
}

type MockSessionService struct{ mock.Mock }

func (m *MockSessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Create(ctx context.Context, userID string, hashedToken string, ipAddress *string, userAgent *string, maxAge time.Duration) (*models.Session, error) {
	args := m.Called(ctx, userID, hashedToken, ipAddress, userAgent, maxAge)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) GetByToken(ctx context.Context, hashedToken string) (*models.Session, error) {
	args := m.Called(ctx, hashedToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionService) Delete(ctx context.Context, ID string) error {
	args := m.Called(ctx, ID)
	return args.Error(0)
}

func (m *MockSessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionService) DeleteAllExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionService) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSessionService) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	args := m.Called(ctx, userID, maxCount)
	return args.Error(0)
}

type MockTokenServiceCore struct{ mock.Mock }

func (m *MockTokenServiceCore) Generate() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTokenServiceCore) Hash(token string) string {
	args := m.Called(token)
	return args.String(0)
}

func (m *MockTokenServiceCore) Encrypt(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenServiceCore) Decrypt(encrypted string) (string, error) {
	args := m.Called(encrypted)
	return args.String(0), args.Error(1)
}
