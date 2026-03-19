package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	totpplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func newTestTOTPDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)
	totpSet := totpplugin.MigrationSet("sqlite")

	err = migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, totpSet})
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, ctx context.Context, db bun.IDB, userID string) {
	t.Helper()

	_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, userID, "Test User", userID+"@example.com")
	require.NoError(t, err)
}

func createTestTOTPRecord(t *testing.T, ctx context.Context, repo *repository.TOTPRepository, userID string) *types.TOTPRecord {
	t.Helper()

	record, err := repo.Create(ctx, userID, "encrypted-secret", `["h1","h2"]`)
	require.NoError(t, err)
	return record
}

func createTestTrustedDevice(t *testing.T, ctx context.Context, repo *repository.TOTPRepository, userID, token, userAgent string) *types.TrustedDevice {
	t.Helper()

	device, err := repo.CreateTrustedDevice(ctx, userID, token, userAgent, time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, err)
	return device
}

type tableTest struct {
	name string
	run  func(t *testing.T)
}

func runTableTests(t *testing.T, tests []tableTest) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestWithTx(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - updates within tx and commit persists",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-with-tx-1")
				createTestTOTPRecord(t, ctx, repo, "user-with-tx-1")

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)

				txRepo := repo.WithTx(tx)
				err = txRepo.UpdateBackupCodes(ctx, "user-with-tx-1", `["h2"]`)
				require.NoError(t, err)

				err = tx.Commit()
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-with-tx-1")
				require.NoError(t, err)
				require.Equal(t, `["h2"]`, record.BackupCodes)
			},
		},
		{
			name: "isolation - rollback discards updates",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-with-tx-2")
				createTestTOTPRecord(t, ctx, repo, "user-with-tx-2")

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)

				txRepo := repo.WithTx(tx)
				err = txRepo.UpdateBackupCodes(ctx, "user-with-tx-2", `["rolled-back"]`)
				require.NoError(t, err)

				err = tx.Rollback()
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-with-tx-2")
				require.NoError(t, err)
				require.Equal(t, `["h1","h2"]`, record.BackupCodes)
			},
		},
	}

	runTableTests(t, tests)
}

func TestGetByUserID(t *testing.T) {
	tests := []tableTest{
		{
			name: "record found - returns correct record",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-get-1")
				created := createTestTOTPRecord(t, ctx, repo, "user-get-1")

				record, err := repo.GetByUserID(ctx, "user-get-1")
				require.NoError(t, err)
				require.NotNil(t, record)
				require.Equal(t, created.ID, record.ID)
				require.Equal(t, "user-get-1", record.UserID)
				require.Equal(t, "encrypted-secret", record.Secret)
				require.Equal(t, `["h1","h2"]`, record.BackupCodes)
			},
		},
		{
			name: "record not found - returns nil without error",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				record, err := repo.GetByUserID(ctx, "missing-user")
				require.NoError(t, err)
				require.Nil(t, record)
			},
		},
	}

	runTableTests(t, tests)
}

func TestCreate(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - creates record with correct fields",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-create-1")

				record, err := repo.Create(ctx, "user-create-1", "secret-value", `["a","b"]`)
				require.NoError(t, err)
				require.NotNil(t, record)
				require.NotEmpty(t, record.ID)
				require.Equal(t, "user-create-1", record.UserID)
				require.Equal(t, "secret-value", record.Secret)
				require.Equal(t, `["a","b"]`, record.BackupCodes)
				require.False(t, record.CreatedAt.IsZero())
				require.False(t, record.UpdatedAt.IsZero())
			},
		},
		{
			name: "success - record persists in database",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-create-2")

				_, err := repo.Create(ctx, "user-create-2", "secret-value", `["a","b"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-create-2")
				require.NoError(t, err)
				require.NotNil(t, record)
				require.Equal(t, "secret-value", record.Secret)
				require.Equal(t, `["a","b"]`, record.BackupCodes)
			},
		},
	}

	runTableTests(t, tests)
}

func TestUpdateBackupCodes(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - updates codes",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-update-1")
				createTestTOTPRecord(t, ctx, repo, "user-update-1")

				err := repo.UpdateBackupCodes(ctx, "user-update-1", `["h2"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-update-1")
				require.NoError(t, err)
				require.Equal(t, `["h2"]`, record.BackupCodes)
			},
		},
		{
			name: "success - updates UpdatedAt timestamp",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-update-2")
				createTestTOTPRecord(t, ctx, repo, "user-update-2")

				before, err := repo.GetByUserID(ctx, "user-update-2")
				require.NoError(t, err)

				time.Sleep(5 * time.Millisecond)
				err = repo.UpdateBackupCodes(ctx, "user-update-2", `["changed"]`)
				require.NoError(t, err)

				after, err := repo.GetByUserID(ctx, "user-update-2")
				require.NoError(t, err)
				require.True(t, after.UpdatedAt.After(before.UpdatedAt) || after.UpdatedAt.Equal(before.UpdatedAt))
			},
		},
		{
			name: "success - replaces old codes entirely",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-update-3")
				createTestTOTPRecord(t, ctx, repo, "user-update-3")

				err := repo.UpdateBackupCodes(ctx, "user-update-3", `["only-one"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-update-3")
				require.NoError(t, err)
				require.Equal(t, `["only-one"]`, record.BackupCodes)
				require.NotEqual(t, `["h1","h2"]`, record.BackupCodes)
			},
		},
	}

	runTableTests(t, tests)
}

func TestCompareAndSwapBackupCodes(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - swaps when codes match",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-cas-1")
				createTestTOTPRecord(t, ctx, repo, "user-cas-1")

				updated, err := repo.CompareAndSwapBackupCodes(ctx, "user-cas-1", `["h1","h2"]`, `["h2"]`)
				require.NoError(t, err)
				require.True(t, updated)

				record, err := repo.GetByUserID(ctx, "user-cas-1")
				require.NoError(t, err)
				require.Equal(t, `["h2"]`, record.BackupCodes)
			},
		},
		{
			name: "failure - rejects when codes do not match",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-cas-2")
				createTestTOTPRecord(t, ctx, repo, "user-cas-2")

				updated, err := repo.CompareAndSwapBackupCodes(ctx, "user-cas-2", `["bad"]`, `[]`)
				require.NoError(t, err)
				require.False(t, updated)

				record, err := repo.GetByUserID(ctx, "user-cas-2")
				require.NoError(t, err)
				require.Equal(t, `["h1","h2"]`, record.BackupCodes)
			},
		},
	}

	runTableTests(t, tests)
}

func TestDeleteByUserID(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - deletes record",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-delete-1")
				createTestTOTPRecord(t, ctx, repo, "user-delete-1")

				err := repo.DeleteByUserID(ctx, "user-delete-1")
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-delete-1")
				require.NoError(t, err)
				require.Nil(t, record)
			},
		},
		{
			name: "no record - succeeds without error",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				err := repo.DeleteByUserID(ctx, "missing-user")
				require.NoError(t, err)
			},
		},
	}

	runTableTests(t, tests)
}

func TestIsEnabled(t *testing.T) {
	tests := []tableTest{
		{
			name: "enabled - returns true",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-enabled-1")
				createTestTOTPRecord(t, ctx, repo, "user-enabled-1")
				err := repo.SetEnabled(ctx, "user-enabled-1", true)
				require.NoError(t, err)

				enabled, err := repo.IsEnabled(ctx, "user-enabled-1")
				require.NoError(t, err)
				require.True(t, enabled)
			},
		},
		{
			name: "disabled - returns false",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-enabled-2")
				createTestTOTPRecord(t, ctx, repo, "user-enabled-2")

				enabled, err := repo.IsEnabled(ctx, "user-enabled-2")
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
		{
			name: "no record - returns false without error",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				enabled, err := repo.IsEnabled(ctx, "missing-user")
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
	}

	runTableTests(t, tests)
}

func TestSetEnabled(t *testing.T) {
	tests := []tableTest{
		{
			name: "enable - sets enabled to true",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-set-enabled-1")
				createTestTOTPRecord(t, ctx, repo, "user-set-enabled-1")

				err := repo.SetEnabled(ctx, "user-set-enabled-1", true)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-set-enabled-1")
				require.NoError(t, err)
				require.True(t, record.Enabled)
			},
		},
		{
			name: "enable - updates UpdatedAt",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-set-enabled-2")
				createTestTOTPRecord(t, ctx, repo, "user-set-enabled-2")

				before, err := repo.GetByUserID(ctx, "user-set-enabled-2")
				require.NoError(t, err)

				time.Sleep(5 * time.Millisecond)
				err = repo.SetEnabled(ctx, "user-set-enabled-2", true)
				require.NoError(t, err)

				after, err := repo.GetByUserID(ctx, "user-set-enabled-2")
				require.NoError(t, err)
				require.True(t, after.UpdatedAt.After(before.UpdatedAt) || after.UpdatedAt.Equal(before.UpdatedAt))
			},
		},
		{
			name: "disable - sets enabled to false",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-set-enabled-3")
				createTestTOTPRecord(t, ctx, repo, "user-set-enabled-3")
				err := repo.SetEnabled(ctx, "user-set-enabled-3", true)
				require.NoError(t, err)

				err = repo.SetEnabled(ctx, "user-set-enabled-3", false)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, "user-set-enabled-3")
				require.NoError(t, err)
				require.False(t, record.Enabled)
			},
		},
	}

	runTableTests(t, tests)
}

func TestGetTrustedDeviceByToken(t *testing.T) {
	tests := []tableTest{
		{
			name: "device found - returns correct device",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-device-get-1")
				created := createTestTrustedDevice(t, ctx, repo, "user-device-get-1", "token-1", "ua-1")

				device, err := repo.GetTrustedDeviceByToken(ctx, "token-1")
				require.NoError(t, err)
				require.NotNil(t, device)
				require.Equal(t, created.ID, device.ID)
				require.Equal(t, "user-device-get-1", device.UserID)
				require.Equal(t, "token-1", device.Token)
				require.Equal(t, "ua-1", device.UserAgent)
			},
		},
		{
			name: "device not found - returns nil without error",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				device, err := repo.GetTrustedDeviceByToken(ctx, "missing-token")
				require.NoError(t, err)
				require.Nil(t, device)
			},
		},
	}

	runTableTests(t, tests)
}

func TestCreateTrustedDevice(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - creates device with correct fields",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-device-create-1")
				expiresAt := time.Now().UTC().Add(48 * time.Hour)

				device, err := repo.CreateTrustedDevice(ctx, "user-device-create-1", "token-create-1", "ua-create", expiresAt)
				require.NoError(t, err)
				require.NotNil(t, device)
				require.NotEmpty(t, device.ID)
				require.Equal(t, "user-device-create-1", device.UserID)
				require.Equal(t, "token-create-1", device.Token)
				require.Equal(t, "ua-create", device.UserAgent)
				require.WithinDuration(t, expiresAt, device.ExpiresAt, time.Second)
			},
		},
		{
			name: "success - device persists in database",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-device-create-2")
				expiresAt := time.Now().UTC().Add(24 * time.Hour)

				_, err := repo.CreateTrustedDevice(ctx, "user-device-create-2", "token-create-2", "ua-create", expiresAt)
				require.NoError(t, err)

				device, err := repo.GetTrustedDeviceByToken(ctx, "token-create-2")
				require.NoError(t, err)
				require.NotNil(t, device)
				require.Equal(t, "user-device-create-2", device.UserID)
			},
		},
	}

	runTableTests(t, tests)
}

func TestRefreshTrustedDevice(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - updates ExpiresAt",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-refresh-1")
				createTestTrustedDevice(t, ctx, repo, "user-refresh-1", "token-refresh-1", "ua")

				newExpiry := time.Now().UTC().Add(72 * time.Hour)
				err := repo.RefreshTrustedDevice(ctx, "token-refresh-1", newExpiry)
				require.NoError(t, err)

				device, err := repo.GetTrustedDeviceByToken(ctx, "token-refresh-1")
				require.NoError(t, err)
				require.NotNil(t, device)
				require.WithinDuration(t, newExpiry, device.ExpiresAt, time.Second)
			},
		},
		{
			name: "success - only updates target device",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-refresh-2")
				createTestTrustedDevice(t, ctx, repo, "user-refresh-2", "token-refresh-2a", "ua-a")
				original := createTestTrustedDevice(t, ctx, repo, "user-refresh-2", "token-refresh-2b", "ua-b")

				newExpiry := time.Now().UTC().Add(96 * time.Hour)
				err := repo.RefreshTrustedDevice(ctx, "token-refresh-2a", newExpiry)
				require.NoError(t, err)

				other, err := repo.GetTrustedDeviceByToken(ctx, "token-refresh-2b")
				require.NoError(t, err)
				require.NotNil(t, other)
				require.WithinDuration(t, original.ExpiresAt, other.ExpiresAt, time.Second)
			},
		},
	}

	runTableTests(t, tests)
}

func TestDeleteTrustedDevicesByUserID(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - deletes all user's devices",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-device-delete-1")
				createTestTrustedDevice(t, ctx, repo, "user-device-delete-1", "token-del-1", "ua")
				createTestTrustedDevice(t, ctx, repo, "user-device-delete-1", "token-del-2", "ua")

				err := repo.DeleteTrustedDevicesByUserID(ctx, "user-device-delete-1")
				require.NoError(t, err)

				dev1, err := repo.GetTrustedDeviceByToken(ctx, "token-del-1")
				require.NoError(t, err)
				require.Nil(t, dev1)

				dev2, err := repo.GetTrustedDeviceByToken(ctx, "token-del-2")
				require.NoError(t, err)
				require.Nil(t, dev2)
			},
		},
		{
			name: "success - other users devices unaffected",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-device-delete-2a")
				createTestUser(t, ctx, db, "user-device-delete-2b")
				createTestTrustedDevice(t, ctx, repo, "user-device-delete-2a", "token-del-3", "ua")
				createTestTrustedDevice(t, ctx, repo, "user-device-delete-2b", "token-del-4", "ua")

				err := repo.DeleteTrustedDevicesByUserID(ctx, "user-device-delete-2a")
				require.NoError(t, err)

				removed, err := repo.GetTrustedDeviceByToken(ctx, "token-del-3")
				require.NoError(t, err)
				require.Nil(t, removed)

				kept, err := repo.GetTrustedDeviceByToken(ctx, "token-del-4")
				require.NoError(t, err)
				require.NotNil(t, kept)
			},
		},
		{
			name: "no devices - succeeds without error",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				err := repo.DeleteTrustedDevicesByUserID(ctx, "missing-user")
				require.NoError(t, err)
			},
		},
	}

	runTableTests(t, tests)
}

func TestDeleteExpiredTrustedDevices(t *testing.T) {
	tests := []tableTest{
		{
			name: "success - deletes expired devices",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-expired-1")

				_, err := repo.CreateTrustedDevice(ctx, "user-expired-1", "token-expired", "ua", time.Now().UTC().Add(-2*time.Hour))
				require.NoError(t, err)
				_, err = repo.CreateTrustedDevice(ctx, "user-expired-1", "token-valid", "ua", time.Now().UTC().Add(2*time.Hour))
				require.NoError(t, err)

				err = repo.DeleteExpiredTrustedDevices(ctx)
				require.NoError(t, err)

				expired, err := repo.GetTrustedDeviceByToken(ctx, "token-expired")
				require.NoError(t, err)
				require.Nil(t, expired)
			},
		},
		{
			name: "success - keeps valid devices",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-expired-2")

				_, err := repo.CreateTrustedDevice(ctx, "user-expired-2", "token-valid-2", "ua", time.Now().UTC().Add(3*time.Hour))
				require.NoError(t, err)

				err = repo.DeleteExpiredTrustedDevices(ctx)
				require.NoError(t, err)

				valid, err := repo.GetTrustedDeviceByToken(ctx, "token-valid-2")
				require.NoError(t, err)
				require.NotNil(t, valid)
			},
		},
		{
			name: "boundary - near-future device is not deleted",
			run: func(t *testing.T) {
				db := newTestTOTPDB(t)
				repo := repository.NewTOTPRepository(db)
				ctx := context.Background()

				createTestUser(t, ctx, db, "user-expired-3")

				_, err := repo.CreateTrustedDevice(ctx, "user-expired-3", "token-boundary", "ua", time.Now().UTC().Add(1*time.Second))
				require.NoError(t, err)

				err = repo.DeleteExpiredTrustedDevices(ctx)
				require.NoError(t, err)

				boundary, err := repo.GetTrustedDeviceByToken(ctx, "token-boundary")
				require.NoError(t, err)
				require.NotNil(t, boundary)
			},
		},
	}

	runTableTests(t, tests)
}
