package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/uptrace/bun"
)

type BunUserPermissionsRepository struct {
	db bun.IDB
}

func NewBunUserPermissionsRepository(db bun.IDB) *BunUserPermissionsRepository {
	return &BunUserPermissionsRepository{db: db}
}

type userPermissionGrantRow struct {
	PermissionID          string     `bun:"permission_id"`
	PermissionKey         string     `bun:"permission_key"`
	PermissionDescription *string    `bun:"permission_description"`
	GrantedByUserID       *string    `bun:"granted_by_user_id"`
	GrantedAt             *time.Time `bun:"granted_at"`
	RoleID                string     `bun:"role_id"`
	RoleName              string     `bun:"role_name"`
}

func (r *BunUserPermissionsRepository) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	var scanned []userPermissionGrantRow
	err := r.db.NewSelect().
		TableExpr("access_control_user_roles acur").
		ColumnExpr("ap.id AS permission_id").
		ColumnExpr("ap.key AS permission_key").
		ColumnExpr("ap.description AS permission_description").
		ColumnExpr("arp.granted_by_user_id AS granted_by_user_id").
		ColumnExpr("arp.granted_at AS granted_at").
		ColumnExpr("acr.id AS role_id").
		ColumnExpr("acr.name AS role_name").
		Join("JOIN access_control_roles acr ON acr.id = acur.role_id").
		Join("JOIN access_control_role_permissions arp ON arp.role_id = acr.id").
		Join("JOIN access_control_permissions ap ON ap.id = arp.permission_id").
		Where("acur.user_id = ?", userID).
		Where("(acur.expires_at IS NULL OR acur.expires_at > CURRENT_TIMESTAMP)").
		OrderExpr("ap.key ASC, acr.name ASC, arp.granted_at ASC").
		Scan(ctx, &scanned)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	if len(scanned) == 0 {
		return []types.UserPermissionInfo{}, nil
	}

	permissionsByID := make(map[string]*types.UserPermissionInfo, len(scanned))
	orderedPermissionIDs := make([]string, 0, len(scanned))

	for _, row := range scanned {
		permission, exists := permissionsByID[row.PermissionID]
		if !exists {
			permission = &types.UserPermissionInfo{
				PermissionID:          row.PermissionID,
				PermissionKey:         row.PermissionKey,
				PermissionDescription: row.PermissionDescription,
				GrantedByUserID:       row.GrantedByUserID,
				GrantedAt:             row.GrantedAt,
				Sources:               []types.PermissionGrantSource{},
			}
			permissionsByID[row.PermissionID] = permission
			orderedPermissionIDs = append(orderedPermissionIDs, row.PermissionID)
		}

		permission.Sources = append(permission.Sources, types.PermissionGrantSource{
			RoleID:          row.RoleID,
			RoleName:        row.RoleName,
			GrantedByUserID: row.GrantedByUserID,
			GrantedAt:       row.GrantedAt,
		})
	}

	permissions := make([]types.UserPermissionInfo, 0, len(orderedPermissionIDs))
	for _, permissionID := range orderedPermissionIDs {
		permissions = append(permissions, *permissionsByID[permissionID])
	}

	return permissions, nil
}

func (r *BunUserPermissionsRepository) HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error) {
	if len(permissionKeys) == 0 {
		return true, nil
	}

	permissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission.PermissionKey] = struct{}{}
	}

	for _, permissionKey := range permissionKeys {
		if _, ok := granted[permissionKey]; !ok {
			return false, nil
		}
	}

	return true, nil
}
