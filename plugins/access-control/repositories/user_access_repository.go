package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type BunUserAccessRepository struct {
	db bun.IDB
}

func NewBunUserAccessRepository(db bun.IDB) *BunUserAccessRepository {
	return &BunUserAccessRepository{db: db}
}

func (r *BunUserAccessRepository) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	var rows []types.UserRoleInfo
	now := time.Now().UTC()
	err := r.db.NewSelect().
		TableExpr("access_control_user_roles acur").
		ColumnExpr("acur.role_id AS role_id").
		ColumnExpr("acr.name AS role_name").
		ColumnExpr("acur.expires_at AS expires_at").
		Join("JOIN access_control_roles acr ON acr.id = acur.role_id").
		Where("acur.user_id = ?", userID).
		Where("acur.expires_at IS NULL OR acur.expires_at > ?", now).
		OrderExpr("acr.name ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	if rows == nil {
		return []types.UserRoleInfo{}, nil
	}
	return rows, nil
}

func (r *BunUserAccessRepository) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	var rows []types.UserPermissionInfo
	now := time.Now().UTC()
	err := r.db.NewSelect().
		TableExpr("access_control_user_roles pur").
		ColumnExpr("DISTINCT pp.id AS permission_id").
		ColumnExpr("pp.key AS permission_key").
		Join("JOIN access_control_role_permissions prp ON prp.role_id = pur.role_id").
		Join("JOIN access_control_permissions pp ON pp.id = prp.permission_id").
		Where("pur.user_id = ?", userID).
		Where("pur.expires_at IS NULL OR pur.expires_at > ?", now).
		OrderExpr("pp.key ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get user effective permissions: %w", err)
	}
	if rows == nil {
		return []types.UserPermissionInfo{}, nil
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

func (r *BunUserAccessRepository) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
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

	result := &types.UserWithRoles{
		User: mapRowToUser(rows[0]),
	}

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

type userWithPermissionRow struct {
	UserID        string `bun:"user_id"`
	UserName      string `bun:"user_name"`
	UserEmail     string `bun:"user_email"`
	EmailVerified bool   `bun:"email_verified"`
	Image         *string
	Metadata      []byte
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	PermissionID  *string   `bun:"permission_id"`
	PermissionKey *string   `bun:"permission_key"`
}

func (r *BunUserAccessRepository) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	var rows []userWithPermissionRow
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
		ColumnExpr("ap.id AS permission_id").
		ColumnExpr("ap.key AS permission_key").
		Join("LEFT JOIN access_control_user_roles aur ON aur.user_id = u.id AND (aur.expires_at IS NULL OR aur.expires_at > ?)", now).
		Join("LEFT JOIN access_control_role_permissions arp ON arp.role_id = aur.role_id").
		Join("LEFT JOIN access_control_permissions ap ON ap.id = arp.permission_id").
		Where("u.id = ?", userID).
		OrderExpr("ap.key ASC").
		Scan(ctx, &rows)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user with permissions: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	result := &types.UserWithPermissions{
		User: mapRowToUser(rows[0]),
	}

	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.PermissionID == nil || *row.PermissionID == "" {
			continue
		}
		if _, ok := seen[*row.PermissionID]; ok {
			continue
		}
		seen[*row.PermissionID] = struct{}{}
		permissionKey := ""
		if row.PermissionKey != nil {
			permissionKey = *row.PermissionKey
		}
		result.Permissions = append(result.Permissions, types.UserPermissionInfo{PermissionID: *row.PermissionID, PermissionKey: permissionKey})
	}

	return result, nil
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

func (r userWithRoleRow) GetUserID() string            { return r.UserID }
func (r userWithRoleRow) GetUserName() string          { return r.UserName }
func (r userWithRoleRow) GetUserEmail() string         { return r.UserEmail }
func (r userWithRoleRow) GetEmailVerified() bool       { return r.EmailVerified }
func (r userWithRoleRow) GetImage() *string            { return r.Image }
func (r userWithRoleRow) GetMetadata() []byte          { return r.Metadata }
func (r userWithRoleRow) GetCreatedAt() time.Time      { return r.CreatedAt }
func (r userWithRoleRow) GetUpdatedAt() time.Time      { return r.UpdatedAt }
func (r userWithPermissionRow) GetUserID() string      { return r.UserID }
func (r userWithPermissionRow) GetUserName() string    { return r.UserName }
func (r userWithPermissionRow) GetUserEmail() string   { return r.UserEmail }
func (r userWithPermissionRow) GetEmailVerified() bool { return r.EmailVerified }
func (r userWithPermissionRow) GetImage() *string      { return r.Image }
func (r userWithPermissionRow) GetMetadata() []byte    { return r.Metadata }
func (r userWithPermissionRow) GetCreatedAt() time.Time {
	return r.CreatedAt
}
func (r userWithPermissionRow) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}
