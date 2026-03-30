package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
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

type userWithRoleRow struct {
	UserID        string `bun:"user_id"`
	UserName      string `bun:"user_name"`
	UserEmail     string `bun:"user_email"`
	EmailVerified bool   `bun:"email_verified"`
	Image         *string
	Metadata      []byte
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	RoleID        *string   `bun:"role_id"`
	RoleName      *string   `bun:"role_name"`
}

func (r *BunUserRolesRepository) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	var rows []userWithRoleRow
	now := time.Now().UTC()
	err := r.db.NewSelect().
		TableExpr("users u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.name AS user_name").
		ColumnExpr("u.email AS user_email").
		ColumnExpr("u.email_verified AS email_verified").
		ColumnExpr("u.image AS image").
		ColumnExpr("u.metadata AS metadata").
		ColumnExpr("u.created_at AS created_at").
		ColumnExpr("u.updated_at AS updated_at").
		ColumnExpr("pr.id AS role_id").
		ColumnExpr("pr.name AS role_name").
		Join("LEFT JOIN access_control_user_roles pur ON pur.user_id = u.id AND (pur.expires_at IS NULL OR pur.expires_at > ?)", now).
		Join("LEFT JOIN access_control_roles pr ON pr.id = pur.role_id").
		Where("u.id = ?", userID).
		OrderExpr("pr.name ASC").
		Scan(ctx, &rows)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user with roles: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	result := &types.UserWithRoles{User: mapRowToUser(rows[0])}
	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.RoleID == nil || *row.RoleID == "" {
			continue
		}
		if _, ok := seen[*row.RoleID]; ok {
			continue
		}
		seen[*row.RoleID] = struct{}{}
		roleName := ""
		if row.RoleName != nil {
			roleName = *row.RoleName
		}
		result.Roles = append(result.Roles, types.UserRoleInfo{RoleID: *row.RoleID, RoleName: roleName})
	}

	return result, nil
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
				return wrapRepositoryError("insert user role", err)
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
	return wrapRepositoryError("assign user role", err)
}

func (r *BunUserRolesRepository) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
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

type userRow interface {
	GetUserID() string
	GetUserName() string
	GetUserEmail() string
	GetEmailVerified() bool
	GetImage() *string
	GetMetadata() []byte
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

func mapRowToUser(row userRow) models.User {
	return models.User{
		ID:            row.GetUserID(),
		Name:          row.GetUserName(),
		Email:         row.GetUserEmail(),
		EmailVerified: row.GetEmailVerified(),
		Image:         row.GetImage(),
		Metadata:      row.GetMetadata(),
		CreatedAt:     row.GetCreatedAt(),
		UpdatedAt:     row.GetUpdatedAt(),
	}
}

func (r userWithRoleRow) GetUserID() string       { return r.UserID }
func (r userWithRoleRow) GetUserName() string     { return r.UserName }
func (r userWithRoleRow) GetUserEmail() string    { return r.UserEmail }
func (r userWithRoleRow) GetEmailVerified() bool  { return r.EmailVerified }
func (r userWithRoleRow) GetImage() *string       { return r.Image }
func (r userWithRoleRow) GetMetadata() []byte     { return r.Metadata }
func (r userWithRoleRow) GetCreatedAt() time.Time { return r.CreatedAt }
func (r userWithRoleRow) GetUpdatedAt() time.Time { return r.UpdatedAt }
