package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type BunUserStateRepository struct {
	db bun.IDB
}

func NewBunUserStateRepository(db bun.IDB) *BunUserStateRepository {
	return &BunUserStateRepository{db: db}
}

func (r *BunUserStateRepository) GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error) {
	row := &types.AdminUserState{}
	err := r.db.NewSelect().Model(row).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}
	return row, nil
}

func (r *BunUserStateRepository) Upsert(ctx context.Context, state *types.AdminUserState) error {
	now := time.Now().UTC()
	_, err := r.db.NewInsert().
		Model(state).
		On("CONFLICT (user_id) DO UPDATE").
		Set("banned = EXCLUDED.banned").
		Set("banned_at = EXCLUDED.banned_at").
		Set("banned_until = EXCLUDED.banned_until").
		Set("banned_reason = EXCLUDED.banned_reason").
		Set("banned_by_user_id = EXCLUDED.banned_by_user_id").
		Set("updated_at = ?", now).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert user state: %w", err)
	}
	return nil
}

func (r *BunUserStateRepository) Delete(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().Model((*types.AdminUserState)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user state: %w", err)
	}
	return nil
}

func (r *BunUserStateRepository) GetBanned(ctx context.Context) ([]types.AdminUserState, error) {
	var rows []types.AdminUserState
	err := r.db.NewSelect().
		Model(&rows).
		Where("banned = ?", true).
		OrderExpr("updated_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get banned user states: %w", err)
	}

	if rows == nil {
		return []types.AdminUserState{}, nil
	}

	return rows, nil
}
