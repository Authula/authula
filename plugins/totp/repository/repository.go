package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
)

type TOTPRepository struct {
	db bun.IDB
}

func NewTOTPRepository(db bun.IDB) *TOTPRepository {
	return &TOTPRepository{db: db}
}

func (r *TOTPRepository) WithTx(tx bun.IDB) *TOTPRepository {
	return &TOTPRepository{db: tx}
}

func (r *TOTPRepository) GetByUserID(ctx context.Context, userID string) (*TOTPRecord, error) {
	record := new(TOTPRecord)
	err := r.db.NewSelect().
		Model(record).
		Where("user_id = ?", userID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return record, err
}

func (r *TOTPRepository) Create(ctx context.Context, userID, secret, backupCodes string) (*TOTPRecord, error) {
	record := &TOTPRecord{
		ID:          util.GenerateUUID(),
		UserID:      userID,
		Secret:      secret,
		BackupCodes: backupCodes,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(record).
			Exec(ctx)
		if err != nil {
			return err
		}

		return tx.NewSelect().
			Model(record).
			WherePK().
			Scan(ctx)
	})

	return record, err
}

func (r *TOTPRepository) UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error {
	_, err := r.db.NewUpdate().
		Model(&TOTPRecord{}).
		Set("backup_codes = ?", backupCodes).
		Set("updated_at = ?", time.Now().UTC()).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TOTPRepository) CompareAndSwapBackupCodes(ctx context.Context, userID, expectedBackupCodes, newBackupCodes string) (bool, error) {
	res, err := r.db.NewUpdate().
		Model(&TOTPRecord{}).
		Set("backup_codes = ?", newBackupCodes).
		Set("updated_at = ?", time.Now().UTC()).
		Where("user_id = ?", userID).
		Where("backup_codes = ?", expectedBackupCodes).
		Exec(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *TOTPRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().
		Model(&TOTPRecord{}).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TOTPRepository) IsEnabled(ctx context.Context, userID string) (bool, error) {
	record := new(TOTPRecord)
	err := r.db.NewSelect().
		Model(record).
		Column("enabled").
		Where("user_id = ?", userID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return record.Enabled, nil
}

func (r *TOTPRepository) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	_, err := r.db.NewUpdate().
		Model(&TOTPRecord{}).
		Set("enabled = ?", enabled).
		Set("updated_at = ?", time.Now().UTC()).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TOTPRepository) GetTrustedDeviceByToken(ctx context.Context, token string) (*TrustedDevice, error) {
	device := new(TrustedDevice)
	err := r.db.NewSelect().
		Model(device).
		Where("token = ?", token).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return device, err
}

func (r *TOTPRepository) CreateTrustedDevice(ctx context.Context, userID, token, userAgent string, expiresAt time.Time) (*TrustedDevice, error) {
	device := &TrustedDevice{
		ID:        util.GenerateUUID(),
		UserID:    userID,
		Token:     token,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(device).
			Exec(ctx)
		if err != nil {
			return err
		}

		return tx.NewSelect().
			Model(device).
			WherePK().
			Scan(ctx)
	})

	return device, err
}

func (r *TOTPRepository) RefreshTrustedDevice(ctx context.Context, token string, expiresAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model(&TrustedDevice{}).
		Set("expires_at = ?", expiresAt).
		Where("token = ?", token).
		Exec(ctx)
	return err
}

func (r *TOTPRepository) DeleteTrustedDevicesByUserID(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().
		Model(&TrustedDevice{}).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TOTPRepository) DeleteExpiredTrustedDevices(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model(&TrustedDevice{}).
		Where("expires_at < ?", time.Now().UTC()).
		Exec(ctx)
	return err
}
