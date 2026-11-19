package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/migrationset"
	"github.com/Authula/authula/plugins/jwt/repositories"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

func newJWTKeyTestService(t *testing.T) (KeyService, SigningKeyService) {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	set := migrations.MigrationSet{
		PluginID:   models.PluginJWT.String(),
		Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
	}
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(context.Background(), []migrations.MigrationSet{set}))

	shared := NewSigningKeyService(
		repositories.NewBunSigningKeyRepository(db),
		jwttests.NopTokenService{},
		&internaltests.MockLogger{},
		SigningKeyConfig{GracePeriod: time.Hour},
	)
	service := NewKeyService(shared, &internaltests.MockLogger{}, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	return service, shared
}

func TestKeyService_UsesSharedPurposeAndAlgorithm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _ := newJWTKeyTestService(t)

	require.NoError(t, service.GenerateKeysIfMissing(ctx))
	require.NoError(t, service.GenerateKeysIfMissing(ctx))

	active, err := service.GetActiveKey(ctx)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, models.SigningKeyPurposeJWTPlugin, active.KeyPurpose)
	assert.Equal(t, models.SigningKeyAlgorithmRS256, active.Algorithm)
	assert.NotEmpty(t, active.KeyID)

	publicKeys, err := service.GetPublicKeys(ctx)
	require.NoError(t, err)
	require.Len(t, publicKeys, 1)
	assert.Equal(t, active.KeyID, publicKeys[0].KeyID)
}

func TestKeyService_RotationPolicy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, shared := newJWTKeyTestService(t)
	require.NoError(t, service.GenerateKeysIfMissing(ctx))

	assert.False(t, service.IsKeyRotationDue(ctx, time.Hour))

	rotated, err := service.RotateKeysIfNeeded(ctx, time.Hour, time.Hour, nil)
	require.NoError(t, err)
	assert.False(t, rotated)

	// Rotate an old key through the shared service, then confirm the plugin
	// policy sees the new key as fresh.
	require.NoError(t, shared.RotateKeyWithGrace(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, time.Minute))
	assert.False(t, service.IsKeyRotationDue(ctx, time.Hour))
}

func TestKeyService_FreshlyRotatedKeyIsNotDue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, shared := newJWTKeyTestService(t)
	require.NoError(t, service.GenerateKeysIfMissing(ctx))

	// Rotate the active key and confirm the plugin sees the new key as fresh.
	active, err := shared.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	_, err = shared.GetPublicKeys(ctx, models.SigningKeyPurposeJWTPlugin)
	require.NoError(t, err)
	require.NoError(t, shared.RotateKeyWithGrace(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, time.Hour))
	assert.NotEqual(t, active.KeyID, mustActiveKeyID(t, service))

	cacheCalls := 0
	rotated, err := service.RotateKeysIfNeeded(ctx, time.Hour, time.Hour, func(context.Context) error {
		cacheCalls++
		return nil
	})
	require.NoError(t, err)
	assert.False(t, rotated)
	assert.Zero(t, cacheCalls)
}

func mustActiveKeyID(t *testing.T, service KeyService) string {
	t.Helper()
	key, err := service.GetActiveKey(context.Background())
	require.NoError(t, err)
	return key.KeyID
}

var _ models.Logger = (*internaltests.MockLogger)(nil)
