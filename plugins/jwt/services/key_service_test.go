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
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

func setupKeyServiceTest(t *testing.T) (KeyService, repositories.JWKSRepository) {
	t.Helper()
	db := internaltests.NewIntegrationTestDB(t)
	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet()
	require.NoError(t, err)

	jwtSet := migrations.MigrationSet{
		PluginID:   models.PluginJWT.String(),
		DependsOn:  []string{migrations.CorePluginID},
		Migrations: migrationset.Migrations(),
	}

	err = migrator.Migrate(context.Background(), []migrations.MigrationSet{coreSet, jwtSet})
	require.NoError(t, err)

	repo := repositories.NewBunJWKSRepository(db)
	logger := &internaltests.MockLogger{}
	coreTokenSvc := jwttests.NopTokenService{}

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
					ID:         "00000000-0000-0000-0000-000000000001",
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
				require.Equal(t, "00000000-0000-0000-0000-000000000001", keys[0].ID)
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
					ID:         "00000000-0000-0000-0000-000000000001",
					PublicKey:  "public-key-1",
					PrivateKey: "private-key-1",
					CreatedAt:  time.Now(),
				})
				require.NoError(t, err)
			},
			wantID: "00000000-0000-0000-0000-000000000001",
		},
		{
			name: "returns most recent key",
			setup: func(ctx context.Context, repo repositories.JWKSRepository) {
				err := repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "00000000-0000-0000-0000-000000000002",
					PublicKey:  "public-key-old",
					PrivateKey: "private-key-old",
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				})
				require.NoError(t, err)

				err = repo.StoreJWKSKey(ctx, &types.JWKS{
					ID:         "00000000-0000-0000-0000-000000000003",
					PublicKey:  "public-key-new",
					PrivateKey: "private-key-new",
					CreatedAt:  time.Now(),
				})
				require.NoError(t, err)
			},
			wantID: "00000000-0000-0000-0000-000000000003",
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
					ID:         "00000000-0000-0000-0000-000000000002",
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
					ID:         "00000000-0000-0000-0000-000000000004",
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
					oldKey, err := repo.GetJWKSKeyByID(ctx, "00000000-0000-0000-0000-000000000002")
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
