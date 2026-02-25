package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UserStateRepository struct {
	db bun.IDB
}

func NewUserStateRepository(db bun.IDB) *UserStateRepository {
	return &UserStateRepository{db: db}
}

func (r *UserStateRepository) GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error) {
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

func (r *UserStateRepository) Upsert(ctx context.Context, state *types.AdminUserState) error {
	now := time.Now().UTC()
	_, err := r.db.NewInsert().
		Model(state).
		On("CONFLICT (user_id) DO UPDATE").
		Set("is_banned = EXCLUDED.is_banned").
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

func (r *UserStateRepository) Delete(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().Model((*types.AdminUserState)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user state: %w", err)
	}
	return nil
}

func (r *UserStateRepository) GetBanned(ctx context.Context) ([]types.AdminUserState, error) {
	var rows []types.AdminUserState
	err := r.db.NewSelect().
		Model(&rows).
		Where("is_banned = ?", true).
		OrderExpr("updated_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get banned user states: %w", err)
	}
	return rows, nil
}
