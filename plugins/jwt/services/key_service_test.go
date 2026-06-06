package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	coreservices "github.com/Authula/authula/services"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/jwt/migrationset"
	"github.com/Authula/authula/plugins/jwt/repositories"
	"github.com/Authula/authula/plugins/jwt/types"
)

type nopTokenService struct{}

func (nopTokenService) Generate() (string, error)                { return "", nil }
func (nopTokenService) Hash(token string) string                 { return token }
func (nopTokenService) Encrypt(token string) (string, error)     { return token, nil }
func (nopTokenService) Decrypt(encrypted string) (string, error) { return encrypted, nil }

func setupKeyServiceTest(t *testing.T) (KeyService, repositories.JWKSRepository) {
	t.Helper()
	db := internaltests.NewSQLiteIntegrationDB(t)

	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)
	err = migrator.Migrate(context.Background(), []migrations.MigrationSet{
		{
			PluginID:   models.PluginJWT.String(),
			Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
		},
	})
	require.NoError(t, err)

	repo := repositories.NewBunJWKSRepository(db)
	logger := &internaltests.MockLogger{}
	coreTokenSvc := nopTokenService{}

	svc := NewKeyService(repo, logger, coreservices.TokenService(coreTokenSvc), "test-secret")
	return svc, repo
}

func TestKeyService_GenerateKeysIfMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(context.Context, repositories.JWKSRepository)
		wantNewID bool
	}{
		{
			name:      "no keys",
			setup:     func(ctx context.Context, repo repositories.JWKSRepository) {},
			wantNewID: true,
		},
		{
			name: "keys exist",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "pre-seeded-key",
					PublicKey:  "pre-seeded-public-key",
					PrivateKey: "pre-seeded-private-key",
				})
				require.NoError(t, err)
			},
			wantNewID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := setupKeyServiceTest(t)
			ctx := context.Background()

			tt.setup(ctx, repo)

			err := svc.GenerateKeysIfMissing(ctx)
			require.NoError(t, err)

			keys, err := repo.GetJWKSKeys(ctx)
			require.NoError(t, err)

			if tt.wantNewID {
				require.Len(t, keys, 1)
				require.NotEmpty(t, keys[0].ID)
				assert.Contains(t, keys[0].PublicKey, "BEGIN PUBLIC KEY")
				assert.Contains(t, keys[0].PrivateKey, "BEGIN PRIVATE KEY")
			} else {
				require.Len(t, keys, 1)
				require.Equal(t, "pre-seeded-key", keys[0].ID)
			}
		})
	}
}

func TestKeyService_GetActiveKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(context.Context, repositories.JWKSRepository)
		wantID  string
		wantErr string
	}{
		{
			name:    "no keys",
			setup:   func(ctx context.Context, repo repositories.JWKSRepository) {},
			wantErr: "no active key found",
		},
		{
			name: "single key",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "single-key",
					PublicKey:  "public-key-1",
					PrivateKey: "private-key-1",
					CreatedAt:  time.Now(),
				})
				require.NoError(t, err)
			},
			wantID: "single-key",
		},
		{
			name: "returns most recent key",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "old-key",
					PublicKey:  "public-key-old",
					PrivateKey: "private-key-old",
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				})
				require.NoError(t, err)

				err = repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "new-key",
					PublicKey:  "public-key-new",
					PrivateKey: "private-key-new",
					CreatedAt:  time.Now(),
				})
				require.NoError(t, err)
			},
			wantID: "new-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := setupKeyServiceTest(t)
			ctx := context.Background()

			tt.setup(ctx, repo)

			active, err := svc.GetActiveKey(ctx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, active)
			} else {
				require.NoError(t, err)
				require.NotNil(t, active)
				require.Equal(t, tt.wantID, active.ID)
			}
		})
	}
}

func TestKeyService_RotateKeysIfNeeded(t *testing.T) {
	t.Parallel()

	rotationInterval := 24 * time.Hour
	gracePeriod := 1 * time.Hour

	tests := []struct {
		name        string
		setup       func(context.Context, repositories.JWKSRepository)
		wantRotated bool
		wantErr     string
	}{
		{
			name: "rotation due",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "old-key",
					PublicKey:  "old-public-key",
					PrivateKey: "old-private-key",
					CreatedAt:  time.Now().Add(-25 * time.Hour),
				})
				require.NoError(t, err)
			},
			wantRotated: true,
		},
		{
			name: "not due",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "recent-key",
					PublicKey:  "recent-public-key",
					PrivateKey: "recent-private-key",
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				})
				require.NoError(t, err)
			},
			wantRotated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := setupKeyServiceTest(t)
			ctx := context.Background()

			tt.setup(ctx, repo)

			rotated, err := svc.RotateKeysIfNeeded(ctx, rotationInterval, gracePeriod, nil)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantRotated, rotated)

				keys, err := repo.GetJWKSKeys(ctx)
				require.NoError(t, err)

				if tt.wantRotated {
					require.Len(t, keys, 2)
					// Old key should have expires_at set
					oldKey, err := repo.GetJWKSKeyByID(ctx, "old-key")
					require.NoError(t, err)
					require.NotNil(t, oldKey)
					require.NotNil(t, oldKey.ExpiresAt)
				} else {
					require.Len(t, keys, 1)
				}
			}
		})
	}
}
