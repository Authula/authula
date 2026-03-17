package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type BunRolePermissionRepository struct {
	db bun.IDB
}

func NewBunRolePermissionRepository(db bun.IDB) *BunRolePermissionRepository {
	return &BunRolePermissionRepository{db: db}
}

func (r *BunRolePermissionRepository) CreateRole(ctx context.Context, role *types.Role) error {
	_, err := r.db.NewInsert().Model(role).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *BunRolePermissionRepository) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	roles := make([]types.Role, 0)
	err := r.db.NewSelect().Model(&roles).Order("created_at ASC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	return roles, nil
}

func (r *BunRolePermissionRepository) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
	role := new(types.Role)
	err := r.db.NewSelect().Model(role).Where("id = ?", roleID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role by id: %w", err)
	}

	return role, nil
}

func (r *BunRolePermissionRepository) UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error) {
	query := r.db.NewUpdate().
		Model((*types.Role)(nil)).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", roleID)

	if name != nil {
		query = query.Set("name = ?", *name)
	}

	if description != nil {
		query = query.Set("description = ?", *description)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to update role: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine updated rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunRolePermissionRepository) DeleteRole(ctx context.Context, roleID string) (bool, error) {
	result, err := r.db.NewDelete().Model((*types.Role)(nil)).Where("id = ?", roleID).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete role: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine deleted rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunRolePermissionRepository) CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.UserRole)(nil)).
		Where("role_id = ?", roleID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count role user assignments: %w", err)
	}

	return count, nil
}

func (r *BunRolePermissionRepository) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	permissions := make([]types.Permission, 0)
	err := r.db.NewSelect().Model(&permissions).Order("created_at ASC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	return permissions, nil
}

func (r *BunRolePermissionRepository) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	permission := new(types.Permission)
	err := r.db.NewSelect().Model(permission).Where("id = ?", permissionID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get permission by id: %w", err)
	}

	return permission, nil
}

func (r *BunRolePermissionRepository) CreatePermission(ctx context.Context, permission *types.Permission) error {
	_, err := r.db.NewInsert().Model(permission).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (r *BunRolePermissionRepository) UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.Permission)(nil)).
		Set("description = ?", *description).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", permissionID).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to update permission: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine updated rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunRolePermissionRepository) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
	result, err := r.db.NewDelete().Model((*types.Permission)(nil)).Where("id = ?", permissionID).Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete permission: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine deleted rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunRolePermissionRepository) CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.RolePermission)(nil)).
		Where("permission_id = ?", permissionID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count permission role assignments: %w", err)
	}

	return count, nil
}

func (r *BunRolePermissionRepository) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	rows := make([]types.UserPermissionInfo, 0)

	err := r.db.NewSelect().
		TableExpr("access_control_role_permissions arp").
		ColumnExpr("ap.id AS permission_id").
		ColumnExpr("ap.key AS permission_key").
		ColumnExpr("ap.description AS permission_description").
		ColumnExpr("arp.granted_by_user_id AS granted_by_user_id").
		ColumnExpr("arp.granted_at AS granted_at").
		Join("JOIN access_control_permissions ap ON ap.id = arp.permission_id").
		Where("arp.role_id = ?", roleID).
		OrderExpr("ap.key ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return rows, nil
}

func (r *BunRolePermissionRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*types.RolePermission)(nil)).Where("role_id = ?", roleID).Exec(ctx); err != nil {
			return fmt.Errorf("failed to clear role permissions: %w", err)
		}

		now := time.Now().UTC()
		for _, permissionID := range permissionIDs {
			rp := &types.RolePermission{
				RoleID:          roleID,
				PermissionID:    permissionID,
				GrantedByUserID: grantedByUserID,
				GrantedAt:       now,
			}
			if _, err := tx.NewInsert().Model(rp).Exec(ctx); err != nil {
				return fmt.Errorf("failed to insert role permission: %w", err)
			}
		}

		return nil
	})
}

func (r *BunRolePermissionRepository) AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	rp := &types.RolePermission{
		RoleID:          roleID,
		PermissionID:    permissionID,
		GrantedByUserID: grantedByUserID,
		GrantedAt:       time.Now().UTC(),
	}

	_, err := r.db.NewInsert().Model(rp).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add role permission: %w", err)
	}

	return nil
}

func (r *BunRolePermissionRepository) RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error {
	_, err := r.db.NewDelete().
		Model((*types.RolePermission)(nil)).
		Where("role_id = ?", roleID).
		Where("permission_id = ?", permissionID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove role permission: %w", err)
	}

	return nil
}

func (r *BunRolePermissionRepository) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
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
				return fmt.Errorf("failed to insert user role: %w", err)
			}
		}

		return nil
	})
}

func (r *BunRolePermissionRepository) AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	ur := &types.UserRole{
		UserID:           userID,
		RoleID:           roleID,
		AssignedByUserID: assignedByUserID,
		AssignedAt:       time.Now().UTC(),
		ExpiresAt:        expiresAt,
	}

	_, err := r.db.NewInsert().Model(ur).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to assign user role: %w", err)
	}

	return nil
}

func (r *BunRolePermissionRepository) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	_, err := r.db.NewDelete().
		Model((*types.UserRole)(nil)).
		Where("user_id = ?", userID).
		Where("role_id = ?", roleID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove user role: %w", err)
	}

	return nil
}
