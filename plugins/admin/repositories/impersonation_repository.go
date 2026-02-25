package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type ImpersonationRepository struct {
	db bun.IDB
}

func NewImpersonationRepository(db bun.IDB) *ImpersonationRepository {
	return &ImpersonationRepository{db: db}
}

func (r *ImpersonationRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	count, err := r.db.NewSelect().Table("users").Where("id = ?", userID).Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

func (r *ImpersonationRepository) CreateImpersonation(ctx context.Context, impersonation *types.Impersonation) error {
	_, err := r.db.NewInsert().Model(impersonation).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create impersonation: %w", err)
	}
	return nil
}

func (r *ImpersonationRepository) GetActiveImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	row := &types.Impersonation{}
	err := r.db.NewSelect().
		Model(row).
		Where("id = ?", impersonationID).
		Where("ended_at IS NULL").
		Where("expires_at > ?", time.Now().UTC()).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active impersonation by id: %w", err)
	}
	return row, nil
}

func (r *ImpersonationRepository) GetLatestActiveImpersonationByActor(ctx context.Context, actorUserID string) (*types.Impersonation, error) {
	row := &types.Impersonation{}
	err := r.db.NewSelect().
		Model(row).
		Where("actor_user_id = ?", actorUserID).
		Where("ended_at IS NULL").
		Where("expires_at > ?", time.Now().UTC()).
		OrderExpr("started_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest active impersonation: %w", err)
	}
	return row, nil
}

func (r *ImpersonationRepository) EndImpersonation(ctx context.Context, impersonationID string, endedByUserID *string) error {
	now := time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model((*types.Impersonation)(nil)).
		Set("ended_at = ?", now).
		Set("ended_by_user_id = ?", endedByUserID).
		Where("id = ?", impersonationID).
		Where("ended_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to end impersonation: %w", err)
	}
	return nil
}

func (r *ImpersonationRepository) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	var rows []types.Impersonation
	err := r.db.NewSelect().
		Model(&rows).
		OrderExpr("started_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get impersonations: %w", err)
	}

	return rows, nil
}

func (r *ImpersonationRepository) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	row := &types.Impersonation{}
	err := r.db.NewSelect().
		Model(row).
		Where("id = ?", impersonationID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get impersonation by id: %w", err)
	}

	return row, nil
}
