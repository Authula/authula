package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
)

type TwoFactorRepository struct {
	db bun.IDB
}

func NewTwoFactorRepository(db bun.IDB) *TwoFactorRepository {
	return &TwoFactorRepository{db: db}
}

func (r *TwoFactorRepository) WithTx(tx bun.IDB) *TwoFactorRepository {
	return &TwoFactorRepository{db: tx}
}

// --- TwoFactorRecord operations ---

func (r *TwoFactorRepository) GetByUserID(ctx context.Context, userID string) (*TwoFactorRecord, error) {
	record := new(TwoFactorRecord)
	err := r.db.NewSelect().
		Model(record).
		Where("user_id = ?", userID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return record, err
}

func (r *TwoFactorRepository) Create(ctx context.Context, userID, secret, backupCodes string) (*TwoFactorRecord, error) {
	record := &TwoFactorRecord{
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

func (r *TwoFactorRepository) UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error {
	_, err := r.db.NewUpdate().
		Model(&TwoFactorRecord{}).
		Set("backup_codes = ?", backupCodes).
		Set("updated_at = ?", time.Now().UTC()).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TwoFactorRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().
		Model(&TwoFactorRecord{}).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

// --- Enabled operations ---

func (r *TwoFactorRepository) IsEnabled(ctx context.Context, userID string) (bool, error) {
	record := new(TwoFactorRecord)
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

func (r *TwoFactorRepository) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	_, err := r.db.NewUpdate().
		Model(&TwoFactorRecord{}).
		Set("enabled = ?", enabled).
		Set("updated_at = ?", time.Now().UTC()).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

// --- TrustedDevice operations ---

func (r *TwoFactorRepository) GetTrustedDeviceByToken(ctx context.Context, token string) (*TrustedDevice, error) {
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

func (r *TwoFactorRepository) CreateTrustedDevice(ctx context.Context, userID, token, userAgent string, expiresAt time.Time) (*TrustedDevice, error) {
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

func (r *TwoFactorRepository) RefreshTrustedDevice(ctx context.Context, token string, expiresAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model(&TrustedDevice{}).
		Set("expires_at = ?", expiresAt).
		Where("token = ?", token).
		Exec(ctx)
	return err
}

func (r *TwoFactorRepository) DeleteTrustedDevicesByUserID(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().
		Model(&TrustedDevice{}).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *TwoFactorRepository) DeleteExpiredTrustedDevices(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model(&TrustedDevice{}).
		Where("expires_at < ?", time.Now().UTC()).
		Exec(ctx)
	return err
}
