package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/access-control/types"
)

type BunRolePermissionRepository struct {
	RolesRepository
	PermissionsRepository
	RolePermissionsRepository
	UserRolesRepository
}

func NewBunRolePermissionRepository(db bun.IDB) *BunRolePermissionRepository {
	return &BunRolePermissionRepository{
		RolesRepository:           NewBunRolesRepository(db),
		PermissionsRepository:     NewBunPermissionsRepository(db),
		RolePermissionsRepository: NewBunRolePermissionsRepository(db),
		UserRolesRepository:       NewBunUserRolesRepository(db),
	}
}

type BunRolePermissionsRepository struct {
	db bun.IDB
}

func NewBunRolePermissionsRepository(db bun.IDB) *BunRolePermissionsRepository {
	return &BunRolePermissionsRepository{db: db}
}

type rolePermissionRow struct {
	PermissionID          string     `bun:"permission_id"`
	PermissionKey         string     `bun:"permission_key"`
	PermissionDescription *string    `bun:"permission_description"`
	GrantedByUserID       *string    `bun:"granted_by_user_id"`
	GrantedAt             *time.Time `bun:"granted_at"`
}

func (r *BunRolePermissionsRepository) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	var scanned []rolePermissionRow
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
		Scan(ctx, &scanned)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	for _, row := range scanned {
		rows = append(rows, types.UserPermissionInfo{
			PermissionID:          row.PermissionID,
			PermissionKey:         row.PermissionKey,
			PermissionDescription: row.PermissionDescription,
			GrantedByUserID:       row.GrantedByUserID,
			GrantedAt:             row.GrantedAt,
		})
	}

	return rows, nil
}

func (r *BunRolePermissionsRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
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
			_, err := tx.NewInsert().Model(rp).Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BunRolePermissionsRepository) AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	rp := &types.RolePermission{
		RoleID:          roleID,
		PermissionID:    permissionID,
		GrantedByUserID: grantedByUserID,
		GrantedAt:       time.Now().UTC(),
	}

	_, err := r.db.NewInsert().Model(rp).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *BunRolePermissionsRepository) RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error {
	_, err := r.db.NewDelete().
		Model((*types.RolePermission)(nil)).
		Where("role_id = ?", roleID).
		Where("permission_id = ?", permissionID).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *BunRolePermissionsRepository) CountRolesByPermission(ctx context.Context, permissionID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.RolePermission)(nil)).
		Where("permission_id = ?", permissionID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count roles by permission: %w", err)
	}
	return count, nil
}
