package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type BunVerificationRepository struct {
	db bun.IDB
}

func NewBunVerificationRepository(db bun.IDB) VerificationRepository {
	return &BunVerificationRepository{db: db}
}

func (r *BunVerificationRepository) GetByID(ctx context.Context, id string) (*models.Verification, error) {
	v := new(models.Verification)
	err := r.db.NewSelect().Model(v).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (r *BunVerificationRepository) GetByToken(ctx context.Context, token string) (*models.Verification, error) {
	v := new(models.Verification)
	err := r.db.NewSelect().Model(v).Where("token = ?", token).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (r *BunVerificationRepository) Create(ctx context.Context, verification *models.Verification) (*models.Verification, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(verification).
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(verification).
			WherePK().
			Scan(ctx)
		return err
	})

	return verification, err
}

func (r *BunVerificationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&models.Verification{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunVerificationRepository) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	_, err := r.db.NewDelete().Model(&models.Verification{}).Where("user_id = ? AND type = ?", userID, vType).Exec(ctx)
	return err
}

func (r *BunVerificationRepository) WithTx(tx bun.IDB) VerificationRepository {
	return &BunVerificationRepository{db: tx}
}
