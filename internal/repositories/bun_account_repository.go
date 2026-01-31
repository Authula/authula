package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type BunAccountRepository struct {
	db bun.IDB
}

func NewBunAccountRepository(db bun.IDB) AccountRepository {
	return &BunAccountRepository{db: db}
}

func (r *BunAccountRepository) GetByID(ctx context.Context, id string) (*models.Account, error) {
	acc := new(models.Account)
	err := r.db.NewSelect().Model(acc).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return acc, err
}

func (r *BunAccountRepository) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	acc := new(models.Account)
	err := r.db.NewSelect().Model(acc).Where("user_id = ?", userID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return acc, err
}

func (r *BunAccountRepository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	acc := new(models.Account)
	err := r.db.NewSelect().
		Model(acc).
		Where("user_id = ? AND provider_id = ?", userID, provider).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return acc, err
}

func (r *BunAccountRepository) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	acc := new(models.Account)
	err := r.db.NewSelect().
		Model(acc).
		Where("provider_id = ? AND account_id = ?", provider, accountID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return acc, err
}

func (r *BunAccountRepository) Create(ctx context.Context, account *models.Account) (*models.Account, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(account).
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(account).
			WherePK().
			Scan(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}
func (r *BunAccountRepository) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(account).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(account).
			WherePK().
			Scan(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (r *BunAccountRepository) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	q := r.db.NewUpdate().
		Model(&models.Account{}).
		Where("user_id = ?", userID)

	q = util.ApplyFieldUpdates(q, fields)

	_, err := q.Exec(ctx)
	return err
}

func (r *BunAccountRepository) WithTx(tx bun.IDB) AccountRepository {
	return &BunAccountRepository{db: tx}
}
