package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunLoginTransactionRepository struct {
	db bun.IDB
}

func NewBunLoginTransactionRepository(db bun.IDB) LoginTransactionRepository {
	return &BunLoginTransactionRepository{db: db}
}

func (r *BunLoginTransactionRepository) Create(ctx context.Context, transaction *types.OAuthLoginTransaction) (*types.OAuthLoginTransaction, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(transaction).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(transaction).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (r *BunLoginTransactionRepository) GetByTransactionHash(ctx context.Context, transactionHash string) (*types.OAuthLoginTransaction, error) {
	transaction := new(types.OAuthLoginTransaction)
	if err := r.db.NewSelect().Model(transaction).Where("transaction_hash = ?", transactionHash).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return transaction, nil
}

func (r *BunLoginTransactionRepository) MarkCompleted(ctx context.Context, transactionHash string, completedAt time.Time) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthLoginTransaction)(nil)).
		Set("completed_at = ?", completedAt.UTC()).
		Where("transaction_hash = ? AND completed_at IS NULL AND consumed_at IS NULL AND expires_at > ?", transactionHash, completedAt.UTC()).
		Exec(ctx)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *BunLoginTransactionRepository) Consume(ctx context.Context, transactionHash string, consumedAt time.Time) (*types.OAuthLoginTransaction, error) {
	var transaction *types.OAuthLoginTransaction
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthLoginTransaction)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where("transaction_hash = ? AND consumed_at IS NULL AND expires_at > ?", transactionHash, consumedAt.UTC()).
			Exec(ctx)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}

		transaction = new(types.OAuthLoginTransaction)
		return tx.NewSelect().Model(transaction).Where("transaction_hash = ?", transactionHash).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (r *BunLoginTransactionRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthLoginTransaction)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunLoginTransactionRepository) WithTx(tx bun.IDB) LoginTransactionRepository {
	return &BunLoginTransactionRepository{db: tx}
}
