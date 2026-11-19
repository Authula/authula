package repositories

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
)

func newClosedSigningKeyTestRepository(t *testing.T) SigningKeyRepository {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	set := migrations.MigrationSet{
		PluginID:   models.PluginJWT.String(),
		Migrations: migrationset.JWTMigrationsForProvider("sqlite"),
	}
	migrator, err := migrations.NewMigrator(db, nil)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(context.Background(), []migrations.MigrationSet{set}))
	require.NoError(t, db.Close())

	return NewBunSigningKeyRepository(db)
}

func TestSigningKeyRepository_DatabaseErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	key := testSigningKey("1", "kid-1", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)

	tests := []struct {
		name    string
		call    func(SigningKeyRepository) error
		wantMsg string
	}{
		{
			name: "GetActiveKey",
			call: func(repo SigningKeyRepository) error {
				_, err := repo.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
				return err
			},
			wantMsg: "failed to get active signing key",
		},
		{
			name: "GetKeyByID",
			call: func(repo SigningKeyRepository) error {
				_, err := repo.GetKeyByID(ctx, "kid-1")
				return err
			},
			wantMsg: "failed to get signing key by key id",
		},
		{
			name: "GetPublicKeys",
			call: func(repo SigningKeyRepository) error {
				_, err := repo.GetPublicKeys(ctx, models.SigningKeyPurposeJWTPlugin, time.Now())
				return err
			},
			wantMsg: "failed to list signing keys",
		},
		{
			name: "Create",
			call: func(repo SigningKeyRepository) error {
				return repo.Create(ctx, key)
			},
			wantMsg: "failed to store signing key",
		},
		{
			name: "Rotate",
			call: func(repo SigningKeyRepository) error {
				return repo.Rotate(ctx, models.SigningKeyPurposeJWTPlugin, key, time.Now().Add(time.Hour))
			},
			wantMsg: "database is closed",
		},
		{
			name: "DeleteExpired",
			call: func(repo SigningKeyRepository) error {
				_, err := repo.DeleteExpired(ctx, time.Now())
				return err
			},
			wantMsg: "failed to delete expired signing keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newClosedSigningKeyTestRepository(t)

			err := tt.call(repo)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestSigningKeyRepository_RotateFailsWhenDeactivationFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, db := newSigningKeyTestRepository(t)

	oldKey := testSigningKey("1", "kid-old", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)
	require.NoError(t, repo.Create(ctx, oldKey))

	_, err := db.NewRaw(
		"CREATE TRIGGER block_signing_key_update BEFORE UPDATE ON jwks " +
			"BEGIN SELECT RAISE(ABORT, 'update blocked'); END",
	).Exec(ctx)
	require.NoError(t, err)

	newKey := testSigningKey("2", "kid-new", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)
	err = repo.Rotate(ctx, models.SigningKeyPurposeJWTPlugin, newKey, time.Now().Add(time.Hour))
	require.ErrorContains(t, err, "failed to deactivate old signing keys")

	active, err := repo.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	assert.Equal(t, "kid-old", active.KeyID)
}

func TestSigningKeyRepository_RotateRollsBackWhenInsertFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, db := newSigningKeyTestRepository(t)

	oldKey := testSigningKey("1", "kid-old", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)
	require.NoError(t, repo.Create(ctx, oldKey))

	// Reusing the primary key makes the insert fail after the deactivation
	// update, so the transaction must roll the update back.
	failingKey := testSigningKey("1", "kid-new", models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, true, nil)

	err := repo.Rotate(ctx, models.SigningKeyPurposeJWTPlugin, failingKey, time.Now().Add(time.Hour))
	require.ErrorContains(t, err, "failed to store signing key")

	active, err := repo.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)
	require.NoError(t, err)
	assert.Equal(t, "kid-old", active.KeyID)
	assert.True(t, active.IsActive)
	assert.Nil(t, active.ExpiresAt)

	count, err := db.NewSelect().Model((*models.SigningKey)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
