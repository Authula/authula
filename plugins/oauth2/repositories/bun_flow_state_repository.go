package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunFlowStateRepository struct {
	db bun.IDB
}

func NewBunFlowStateRepository(db bun.IDB) FlowStateRepository {
	return &BunFlowStateRepository{db: db}
}

func (r *BunFlowStateRepository) Create(ctx context.Context, state *types.OAuthFlowState) (*types.OAuthFlowState, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(state).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(state).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (r *BunFlowStateRepository) GetByStateHash(ctx context.Context, stateHash string) (*types.OAuthFlowState, error) {
	state := new(types.OAuthFlowState)
	if err := r.db.NewSelect().Model(state).Where("state_hash = ?", stateHash).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return state, nil
}

func (r *BunFlowStateRepository) Consume(ctx context.Context, stateHash string, provider string, purpose string, consumedAt time.Time) (*types.OAuthFlowState, error) {
	var state *types.OAuthFlowState
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthFlowState)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where(
				"state_hash = ? AND provider = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?",
				stateHash,
				provider,
				purpose,
				consumedAt.UTC(),
			).
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

		state = new(types.OAuthFlowState)
		return tx.NewSelect().Model(state).Where("state_hash = ?", stateHash).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (r *BunFlowStateRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthFlowState)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunFlowStateRepository) WithTx(tx bun.IDB) FlowStateRepository {
	return &BunFlowStateRepository{db: tx}
}
