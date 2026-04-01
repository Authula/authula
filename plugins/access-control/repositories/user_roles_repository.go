package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/access-control/types"
)

type BunUserRolesRepository struct {
	db bun.IDB
}

func NewBunUserRolesRepository(db bun.IDB) *BunUserRolesRepository {
	return &BunUserRolesRepository{db: db}
}

func (r *BunUserRolesRepository) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	var rows []types.UserRoleInfo
	err := r.db.NewSelect().
		TableExpr("access_control_user_roles acur").
		ColumnExpr("acur.role_id AS role_id").
		ColumnExpr("acr.name AS role_name").
		ColumnExpr("acr.description AS role_description").
		ColumnExpr("acur.assigned_by_user_id AS assigned_by_user_id").
		ColumnExpr("acur.assigned_at AS assigned_at").
		ColumnExpr("acur.expires_at AS expires_at").
		Join("JOIN access_control_roles acr ON acr.id = acur.role_id").
		Where("acur.user_id = ?", userID).
		OrderExpr("acr.name ASC, acur.assigned_at DESC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	if rows == nil {
		return []types.UserRoleInfo{}, nil
	}
	return rows, nil
}

func (r *BunUserRolesRepository) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*types.UserRole)(nil)).Where("user_id = ?", userID).Exec(ctx); err != nil {
			return fmt.Errorf("failed to clear user roles: %w", err)
		}

		now := time.Now().UTC()
		for _, roleID := range roleIDs {
			ur := &types.UserRole{
				UserID:           userID,
				RoleID:           roleID,
				AssignedByUserID: assignedByUserID,
				AssignedAt:       now,
			}
			if _, err := tx.NewInsert().Model(ur).Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BunUserRolesRepository) AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	ur := &types.UserRole{
		UserID:           userID,
		RoleID:           roleID,
		AssignedByUserID: assignedByUserID,
		AssignedAt:       time.Now().UTC(),
		ExpiresAt:        expiresAt,
	}

	_, err := r.db.NewInsert().Model(ur).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *BunUserRolesRepository) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	_, err := r.db.NewDelete().
		Model((*types.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("role_id = ?", roleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *BunUserRolesRepository) CountUsersByRole(ctx context.Context, roleID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.UserRole)(nil)).
		Where("role_id = ?", roleID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	return count, nil
}
