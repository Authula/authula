package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/access-control/types"
)

type BunRolesRepository struct {
	db bun.IDB
}

func NewBunRolesRepository(db bun.IDB) *BunRolesRepository {
	return &BunRolesRepository{db: db}
}

func (r *BunRolesRepository) CreateRole(ctx context.Context, role *types.Role) error {
	_, err := r.db.NewInsert().Model(role).Exec(ctx)
	return wrapRepositoryError("create role", err)
}

func (r *BunRolesRepository) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	roles := make([]types.Role, 0)
	err := r.db.NewSelect().Model(&roles).Order("created_at ASC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	return roles, nil
}

func (r *BunRolesRepository) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
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

func (r *BunRolesRepository) UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error) {
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
		return false, wrapRepositoryError("update role", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine updated rows: %w", err)
	}

	return affected > 0, nil
}

func (r *BunRolesRepository) DeleteRole(ctx context.Context, roleID string) (bool, error) {
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
