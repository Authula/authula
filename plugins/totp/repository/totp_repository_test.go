package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	totpplugin "github.com/Authula/authula/plugins/totp"
	"github.com/Authula/authula/plugins/totp/repository"
	"github.com/Authula/authula/plugins/totp/types"
)

func newTestTOTPDB(t *testing.T) *bun.DB {
	t.Helper()

	db := internaltests.NewIntegrationTestDB(t)

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet()
	require.NoError(t, err)
	totpSet := totpplugin.MigrationSet()

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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)

				txRepo := repo.WithTx(tx)
				err = txRepo.UpdateBackupCodes(ctx, userID, `["h2"]`)
				require.NoError(t, err)

				err = tx.Commit()
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)

				txRepo := repo.WithTx(tx)
				err = txRepo.UpdateBackupCodes(ctx, userID, `["rolled-back"]`)
				require.NoError(t, err)

				err = tx.Rollback()
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				created := createTestTOTPRecord(t, ctx, repo, userID)

				record, err := repo.GetByUserID(ctx, userID)
				require.NoError(t, err)
				require.NotNil(t, record)
				require.Equal(t, created.ID, record.ID)
				require.Equal(t, userID, record.UserID)
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

				record, err := repo.GetByUserID(ctx, "00000000-0000-0000-0000-000000000000")
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)

				record, err := repo.Create(ctx, userID, "secret-value", `["a","b"]`)
				require.NoError(t, err)
				require.NotNil(t, record)
				require.NotEmpty(t, record.ID)
				require.Equal(t, userID, record.UserID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)

				_, err := repo.Create(ctx, userID, "secret-value", `["a","b"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				err := repo.UpdateBackupCodes(ctx, userID, `["h2"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				before, err := repo.GetByUserID(ctx, userID)
				require.NoError(t, err)

				time.Sleep(5 * time.Millisecond)
				err = repo.UpdateBackupCodes(ctx, userID, `["changed"]`)
				require.NoError(t, err)

				after, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				err := repo.UpdateBackupCodes(ctx, userID, `["only-one"]`)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				updated, err := repo.CompareAndSwapBackupCodes(ctx, userID, `["h1","h2"]`, `["h2"]`)
				require.NoError(t, err)
				require.True(t, updated)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				updated, err := repo.CompareAndSwapBackupCodes(ctx, userID, `["bad"]`, `[]`)
				require.NoError(t, err)
				require.False(t, updated)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				err := repo.DeleteByUserID(ctx, userID)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				err := repo.DeleteByUserID(ctx, "00000000-0000-0000-0000-000000000000")
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)
				err := repo.SetEnabled(ctx, userID, true)
				require.NoError(t, err)

				enabled, err := repo.IsEnabled(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				enabled, err := repo.IsEnabled(ctx, userID)
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

				enabled, err := repo.IsEnabled(ctx, "00000000-0000-0000-0000-000000000000")
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				err := repo.SetEnabled(ctx, userID, true)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)

				before, err := repo.GetByUserID(ctx, userID)
				require.NoError(t, err)

				time.Sleep(5 * time.Millisecond)
				err = repo.SetEnabled(ctx, userID, true)
				require.NoError(t, err)

				after, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTOTPRecord(t, ctx, repo, userID)
				err := repo.SetEnabled(ctx, userID, true)
				require.NoError(t, err)

				err = repo.SetEnabled(ctx, userID, false)
				require.NoError(t, err)

				record, err := repo.GetByUserID(ctx, userID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				created := createTestTrustedDevice(t, ctx, repo, userID, "token-1", "ua-1")

				device, err := repo.GetTrustedDeviceByToken(ctx, "token-1")
				require.NoError(t, err)
				require.NotNil(t, device)
				require.Equal(t, created.ID, device.ID)
				require.Equal(t, userID, device.UserID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				expiresAt := time.Now().UTC().Add(48 * time.Hour)

				device, err := repo.CreateTrustedDevice(ctx, userID, "token-create-1", "ua-create", expiresAt)
				require.NoError(t, err)
				require.NotNil(t, device)
				require.NotEmpty(t, device.ID)
				require.Equal(t, userID, device.UserID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				expiresAt := time.Now().UTC().Add(24 * time.Hour)

				_, err := repo.CreateTrustedDevice(ctx, userID, "token-create-2", "ua-create", expiresAt)
				require.NoError(t, err)

				device, err := repo.GetTrustedDeviceByToken(ctx, "token-create-2")
				require.NoError(t, err)
				require.NotNil(t, device)
				require.Equal(t, userID, device.UserID)
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTrustedDevice(t, ctx, repo, userID, "token-refresh-1", "ua")

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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTrustedDevice(t, ctx, repo, userID, "token-refresh-2a", "ua-a")
				original := createTestTrustedDevice(t, ctx, repo, userID, "token-refresh-2b", "ua-b")

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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)
				createTestTrustedDevice(t, ctx, repo, userID, "token-del-1", "ua")
				createTestTrustedDevice(t, ctx, repo, userID, "token-del-2", "ua")

				err := repo.DeleteTrustedDevicesByUserID(ctx, userID)
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

				userID1 := uuid.New().String()
				userID2 := uuid.New().String()
				createTestUser(t, ctx, db, userID1)
				createTestUser(t, ctx, db, userID2)
				createTestTrustedDevice(t, ctx, repo, userID1, "token-del-3", "ua")
				createTestTrustedDevice(t, ctx, repo, userID2, "token-del-4", "ua")

				err := repo.DeleteTrustedDevicesByUserID(ctx, userID1)
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

				err := repo.DeleteTrustedDevicesByUserID(ctx, "00000000-0000-0000-0000-000000000000")
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)

				_, err := repo.CreateTrustedDevice(ctx, userID, "token-expired", "ua", time.Now().UTC().Add(-2*time.Hour))
				require.NoError(t, err)
				_, err = repo.CreateTrustedDevice(ctx, userID, "token-valid", "ua", time.Now().UTC().Add(2*time.Hour))
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)

				_, err := repo.CreateTrustedDevice(ctx, userID, "token-valid-2", "ua", time.Now().UTC().Add(3*time.Hour))
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

				userID := uuid.New().String()
				createTestUser(t, ctx, db, userID)

				_, err := repo.CreateTrustedDevice(ctx, userID, "token-boundary", "ua", time.Now().UTC().Add(1*time.Second))
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
