package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/migrationset"
)

func newSigningKeyTestRepository(t *testing.T) (SigningKeyRepository, *bun.DB) {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	set := migrations.MigrationSet{
		PluginID:   models.PluginJWT.String(),
		Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
	}
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(context.Background(), []migrations.MigrationSet{set}))

	return NewBunSigningKeyRepository(db), db
}

func testSigningKey(id, kid, purpose, algorithm string, active bool, expiresAt *time.Time) *models.SigningKey {
	return &models.SigningKey{
		ID:         id,
		KeyID:      kid,
		Algorithm:  algorithm,
		KeyPurpose: purpose,
		PublicKey:  "public-" + kid,
		PrivateKey: "private-" + kid,
		IsActive:   active,
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  expiresAt,
	}
}

func TestSigningKeyRepository_GetActiveKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-inactive", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, false, nil)))
	require.NoError(t, repo.Create(ctx, testSigningKey("2", "kid-expired", models.SigningKeyPurposeOIDCAccessToken, models.SigningKeyAlgorithmRS256, true, &past)))
	require.NoError(t, repo.Create(ctx, testSigningKey("3", "kid-active", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, &future)))
	require.NoError(t, repo.Create(ctx, testSigningKey("4", "kid-other-purpose", models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256, true, nil)))
	require.NoError(t, repo.Create(ctx, testSigningKey("5", "kid-other-algorithm", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmES256, true, nil)))

	key, err := repo.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	assert.Equal(t, "kid-active", key.KeyID)

	_, err = repo.GetActiveKey(ctx, models.SigningKeyPurposeOIDCAccessToken, models.SigningKeyAlgorithmRS256)
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)
}

func TestSigningKeyRepository_GetKeyByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-1", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)))

	key, err := repo.GetKeyByID(ctx, "kid-1")
	require.NoError(t, err)
	assert.Equal(t, "1", key.ID)

	_, err = repo.GetKeyByID(ctx, "missing")
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)
}

func TestSigningKeyRepository_GetPublicKeysFiltersPurposeAndExpiry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-active", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)))
	require.NoError(t, repo.Create(ctx, testSigningKey("2", "kid-retired", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, false, &future)))
	require.NoError(t, repo.Create(ctx, testSigningKey("3", "kid-expired", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, false, &past)))
	require.NoError(t, repo.Create(ctx, testSigningKey("4", "kid-other", models.SigningKeyPurposeOIDCIDToken, models.SigningKeyAlgorithmRS256, true, nil)))

	keys, err := repo.GetPublicKeys(ctx, models.SigningKeyPurposeJWTPlugin, time.Now())
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "kid-active", keys[0].KeyID)
	assert.Equal(t, "kid-retired", keys[1].KeyID)
}

func TestSigningKeyRepository_CreateRejectsDuplicateActiveKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-1", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)))

	err := repo.Create(ctx, testSigningKey("2", "kid-2", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil))
	require.Error(t, err)

	// A different algorithm for the same purpose is allowed.
	require.NoError(t, repo.Create(ctx, testSigningKey("3", "kid-3", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmES256, true, nil)))
}

func TestSigningKeyRepository_RotateDeactivatesPreviousKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-old", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)))

	expiresAt := time.Now().Add(time.Hour)
	newKey := testSigningKey("2", "kid-new", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)
	require.NoError(t, repo.Rotate(ctx, models.SigningKeyPurposeJWTPlugin, newKey, expiresAt))

	active, err := repo.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	assert.Equal(t, "kid-new", active.KeyID)

	old, err := repo.GetKeyByID(ctx, "kid-old")
	require.NoError(t, err)
	assert.False(t, old.IsActive)
	require.NotNil(t, old.ExpiresAt)
	assert.WithinDuration(t, expiresAt, *old.ExpiresAt, time.Second)
}

func TestSigningKeyRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, _ := newSigningKeyTestRepository(t)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	require.NoError(t, repo.Create(ctx, testSigningKey("1", "kid-expired", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, false, &past)))
	require.NoError(t, repo.Create(ctx, testSigningKey("2", "kid-live", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmES256, true, &future)))

	deleted, err := repo.DeleteExpired(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.GetKeyByID(ctx, "kid-expired")
	require.ErrorIs(t, err, models.ErrSigningKeyNotFound)

	_, err = repo.GetKeyByID(ctx, "kid-live")
	require.NoError(t, err)
}
