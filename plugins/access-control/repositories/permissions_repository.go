package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/access-control/types"
)

type BunPermissionsRepository struct {
	db bun.IDB
}

func NewBunPermissionsRepository(db bun.IDB) *BunPermissionsRepository {
	return &BunPermissionsRepository{db: db}
}

func (r *BunPermissionsRepository) CreatePermission(ctx context.Context, permission *types.Permission) error {
	_, err := r.db.NewInsert().Model(permission).Exec(ctx)
	return wrapRepositoryError("create permission", err)
}

func (r *BunPermissionsRepository) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	permissions := make([]types.Permission, 0)
	err := r.db.NewSelect().Model(&permissions).Order("created_at ASC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	return permissions, nil
}

func (r *BunPermissionsRepository) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
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

func (r *BunPermissionsRepository) UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error) {
	query := r.db.NewUpdate().
		Model((*types.Permission)(nil)).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", permissionID)

	if description != nil {
		query = query.Set("description = ?", *description)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return false, wrapRepositoryError("update permission", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine updated rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunPermissionsRepository) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
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
