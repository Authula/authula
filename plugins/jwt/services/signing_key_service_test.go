package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/migrationset"
	"github.com/Authula/authula/plugins/jwt/repositories"
)

type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func newJWKTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(dsn))
	require.NoError(t, err)
	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type plainEncryptor struct{}

func (plainEncryptor) Encrypt(value string) (string, error) { return value, nil }

func newTestService(t *testing.T, config SigningKeyConfig) (SigningKeyService, repositories.SigningKeyRepository, *bun.DB) {
	t.Helper()

	db := newJWKTestDB(t)
	set := migrations.MigrationSet{
		PluginID:   models.PluginJWT.String(),
		Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
	}
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(context.Background(), []migrations.MigrationSet{set}))

	repo := repositories.NewBunSigningKeyRepository(db)
	service := NewSigningKeyService(repo, plainEncryptor{}, nopLogger{}, config)
	return service, repo, db
}

func TestSigningKeyService_GeneratesSupportedAlgorithms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		algorithm string
		check     func(*testing.T, any)
	}{
		{
			name:      "RS256 uses an RSA key",
			algorithm: models.SigningKeyAlgorithmRS256,
			check: func(t *testing.T, publicKey any) {
				key, ok := publicKey.(*rsa.PublicKey)
				require.True(t, ok)
				assert.GreaterOrEqual(t, key.N.BitLen(), models.DefaultRSAKeySize)
			},
		},
		{
			name:      "PS256 uses an RSA key",
			algorithm: models.SigningKeyAlgorithmPS256,
			check: func(t *testing.T, publicKey any) {
				key, ok := publicKey.(*rsa.PublicKey)
				require.True(t, ok)
				assert.GreaterOrEqual(t, key.N.BitLen(), models.DefaultRSAKeySize)
			},
		},
		{
			name:      "ES256 uses a P-256 key",
			algorithm: models.SigningKeyAlgorithmES256,
			check: func(t *testing.T, publicKey any) {
				key, ok := publicKey.(*ecdsa.PublicKey)
				require.True(t, ok)
				assert.Equal(t, "P-256", key.Curve.Params().Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			service, _, _ := newTestService(t, SigningKeyConfig{})

			require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, tt.algorithm))

			active, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, tt.algorithm)
			require.NoError(t, err)
			assert.Equal(t, tt.algorithm, active.Algorithm)
			assert.Equal(t, models.SigningKeyPurposeJWTPlugin, active.KeyPurpose)
			assert.NotEmpty(t, active.KeyID)
			assert.True(t, active.IsActive)

			block, _ := pem.Decode([]byte(active.PublicKey))
			require.NotNil(t, block)
			assert.Equal(t, "PUBLIC KEY", block.Type)
			publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
			require.NoError(t, err)
			tt.check(t, publicKey)

			privateBlock, _ := pem.Decode([]byte(active.PrivateKey))
			require.NotNil(t, privateBlock)
			assert.Equal(t, "PRIVATE KEY", privateBlock.Type)
			_, err = x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
			require.NoError(t, err)
		})
	}
}

func TestSigningKeyService_RejectsUnsupportedAlgorithmsAndPurposes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, SigningKeyConfig{})

	for _, algorithm := range []string{"EdDSA", "ES384", "HS256", ""} {
		t.Run("generate_"+algorithm, func(t *testing.T) {
			err := service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, algorithm)
			require.ErrorIs(t, err, models.ErrUnsupportedSigningAlgorithm)
		})
		t.Run("rotate_"+algorithm, func(t *testing.T) {
			err := service.RotateKey(ctx, models.SigningKeyPurposeJWTPlugin, algorithm)
			require.ErrorIs(t, err, models.ErrUnsupportedSigningAlgorithm)
		})
		t.Run("get_active_"+algorithm, func(t *testing.T) {
			_, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, algorithm)
			require.ErrorIs(t, err, models.ErrUnsupportedSigningAlgorithm)
		})
	}

	_, err := service.GetActiveKey(ctx, "unknown", models.SigningKeyAlgorithmRS256)
	require.ErrorIs(t, err, models.ErrInvalidSigningKeyPurpose)
}

func TestSigningKeyService_IsolatesKeysByPurpose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, SigningKeyConfig{})

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256))
	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmES256))

	jwtKey, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	oidcKey, err := service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmES256)
	require.NoError(t, err)

	assert.NotEqual(t, jwtKey.KeyID, oidcKey.KeyID)
	assert.Equal(t, models.SigningKeyAlgorithmRS256, jwtKey.Algorithm)
	assert.Equal(t, models.SigningKeyAlgorithmES256, oidcKey.Algorithm)

	jwtPublic, err := service.GetPublicKeys(ctx, models.SigningKeyPurposeJWTPlugin)
	require.NoError(t, err)
	require.Len(t, jwtPublic, 1)
	assert.Equal(t, jwtKey.KeyID, jwtPublic[0].KeyID)
}

func TestSigningKeyService_GenerateKeyIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, SigningKeyConfig{})

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256))
	first, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256))
	second, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)

	assert.Equal(t, first.KeyID, second.KeyID)
}

func TestSigningKeyService_ConcurrentGenerationCreatesOneActiveKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, db := newTestService(t, SigningKeyConfig{})

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
		}()
	}
	wg.Wait()
	close(errs)

	succeeded := 0
	for err := range errs {
		if err == nil {
			succeeded++
		}
	}
	require.Positive(t, succeeded)

	count, err := db.NewSelect().
		Model((*models.SigningKey)(nil)).
		Where("key_purpose = ? AND algorithm = ? AND is_active = ?", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	active, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	assert.NotEmpty(t, active.KeyID)
}

func TestSigningKeyService_RotationGracePeriodCoversPerClientLifetime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		gracePeriod time.Duration
		maxLifetime time.Duration
		clockSkew   time.Duration
		wantAbout   time.Duration
	}{
		{
			name:        "uses configured grace when it is longer",
			gracePeriod: 2 * time.Hour,
			maxLifetime: 30 * time.Minute,
			clockSkew:   5 * time.Minute,
			wantAbout:   2*time.Hour + 5*time.Minute,
		},
		{
			name:        "extends grace to cover a longer client lifetime",
			gracePeriod: time.Minute,
			maxLifetime: 45 * time.Minute,
			clockSkew:   5 * time.Minute,
			wantAbout:   50 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			service, _, _ := newTestService(t, SigningKeyConfig{
				GracePeriod: tt.gracePeriod,
				ClockSkew:   tt.clockSkew,
				MaxTokenLifetime: func(context.Context) (time.Duration, error) {
					return tt.maxLifetime, nil
				},
			})

			require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256))
			oldKey, err := service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256)
			require.NoError(t, err)

			require.NoError(t, service.RotateKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256))

			publicKeys, err := service.GetPublicKeys(ctx, models.SigningKeyPurposeOIDCIDToken)
			require.NoError(t, err)
			require.Len(t, publicKeys, 2)

			var retired *models.SigningKey
			for i := range publicKeys {
				if publicKeys[i].KeyID == oldKey.KeyID {
					retired = &publicKeys[i]
				}
			}
			require.NotNil(t, retired)
			require.NotNil(t, retired.ExpiresAt)
			assert.WithinDuration(t, time.Now().Add(tt.wantAbout), *retired.ExpiresAt, 5*time.Second)

			active, err := service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256)
			require.NoError(t, err)
			assert.NotEqual(t, oldKey.KeyID, active.KeyID)
		})
	}
}

func TestSigningKeyService_AlgorithmChangeKeepsOldKeysUntilExpiry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, db := newTestService(t, SigningKeyConfig{GracePeriod: time.Hour})

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256))
	oldKey, err := service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)

	require.NoError(t, service.RotateKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmES256))

	active, err := service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmES256)
	require.NoError(t, err)
	assert.Equal(t, models.SigningKeyAlgorithmES256, active.Algorithm)

	_, err = service.GetActiveKey(ctx, models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256)
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)

	publicKeys, err := service.GetPublicKeys(ctx, models.SigningKeyPurposeOIDCIDToken)
	require.NoError(t, err)
	require.Len(t, publicKeys, 2)
	assert.Equal(t, oldKey.KeyID, publicKeys[0].KeyID)
	require.NotNil(t, publicKeys[0].ExpiresAt)

	// Once the old key expires it is no longer published.
	past := time.Now().Add(-time.Minute)
	_, err = db.NewUpdate().Model((*models.SigningKey)(nil)).Set("expires_at = ?", past).Where("key_id = ?", oldKey.KeyID).Exec(ctx)
	require.NoError(t, err)

	publicKeys, err = service.GetPublicKeys(ctx, models.SigningKeyPurposeOIDCIDToken)
	require.NoError(t, err)
	require.Len(t, publicKeys, 1)
	assert.Equal(t, active.KeyID, publicKeys[0].KeyID)
}

func TestSigningKeyService_GetPublicKeysClearsPrivateKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, SigningKeyConfig{})

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256))

	keys, err := service.GetPublicKeys(ctx, models.SigningKeyPurposeJWTPlugin)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Empty(t, keys[0].PrivateKey)
	assert.NotEmpty(t, keys[0].PublicKey)
}

func TestSigningKeyService_GetKeyByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, SigningKeyConfig{})

	require.NoError(t, service.GenerateKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256))
	active, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)

	byID, err := service.GetKeyByID(ctx, active.KeyID)
	require.NoError(t, err)
	assert.Equal(t, active.ID, byID.ID)

	_, err = service.GetKeyByID(ctx, "missing")
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)

	_, err = service.GetKeyByID(ctx, "")
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)
}
