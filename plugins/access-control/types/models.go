package types

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// Models

type Role struct {
	bun.BaseModel `bun:"table:access_control_roles"`

	ID          string    `json:"id" bun:"column:id,pk"`
	Name        string    `json:"name" bun:"column:name"`
	Description *string   `json:"description" bun:"column:description"`
	IsSystem    bool      `json:"is_system" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type Permission struct {
	bun.BaseModel `bun:"table:access_control_permissions"`

	ID          string    `json:"id" bun:"column:id,pk"`
	Key         string    `json:"key" bun:"column:key"`
	Description *string   `json:"description" bun:"column:description"`
	IsSystem    bool      `json:"is_system" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type RolePermission struct {
	bun.BaseModel `bun:"table:access_control_role_permissions"`

	RoleID          string    `json:"role_id" bun:"column:role_id,pk"`
	PermissionID    string    `json:"permission_id" bun:"column:permission_id,pk"`
	GrantedByUserID *string   `json:"granted_by_user_id" bun:"column:granted_by_user_id"`
	GrantedAt       time.Time `json:"granted_at" bun:"column:granted_at"`
}

type UserRole struct {
	bun.BaseModel `bun:"table:access_control_user_roles"`

	UserID           string     `json:"user_id" bun:"column:user_id,pk"`
	RoleID           string     `json:"role_id" bun:"column:role_id,pk"`
	AssignedByUserID *string    `json:"assigned_by_user_id" bun:"column:assigned_by_user_id"`
	AssignedAt       time.Time  `json:"assigned_at" bun:"column:assigned_at"`
	ExpiresAt        *time.Time `json:"expires_at" bun:"column:expires_at"`
}

// Types

type CreateRoleRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsSystem    bool    `json:"is_system"`
}

type CreateRoleResponse struct {
	Role *Role `json:"role"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type UpdateRoleResponse struct {
	Role *Role `json:"role"`
}

type DeleteRoleResponse struct {
	Message string `json:"message"`
}

type CreatePermissionRequest struct {
	Key         string  `json:"key"`
	Description *string `json:"description,omitempty"`
	IsSystem    bool    `json:"is_system"`
}

type CreatePermissionResponse struct {
	Permission *Permission `json:"permission"`
}

type UpdatePermissionRequest struct {
	Description *string `json:"description,omitempty"`
}

type UpdatePermissionResponse struct {
	Permission *Permission `json:"permission"`
}

type DeletePermissionResponse struct {
	Message string `json:"message"`
}

type AddRolePermissionRequest struct {
	PermissionID string `json:"permission_id"`
}

type AddRolePermissionResponse struct {
	Message string `json:"message"`
}

type ReplaceRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

type ReplaceRolePermissionResponse struct {
	Message string `json:"message"`
}

type RemoveRolePermissionResponse struct {
	Message string `json:"message"`
}

type AssignUserRoleRequest struct {
	RoleID    string     `json:"role_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ReplaceUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type ReplaceUserRolesResponse struct {
	Message string `json:"message"`
}

type AssignUserRoleResponse struct {
	Message string `json:"message"`
}

type RemoveUserRoleResponse struct {
	Message string `json:"message"`
}

type GetUserEffectivePermissionsResponse struct {
	Permissions []UserPermissionInfo `json:"permissions"`
}

type UserRoleInfo struct {
	RoleID           string     `json:"role_id"`
	RoleName         string     `json:"role_name"`
	RoleDescription  *string    `json:"role_description,omitempty"`
	AssignedByUserID *string    `json:"assigned_by_user_id,omitempty"`
	AssignedAt       *time.Time `json:"assigned_at,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

type PermissionGrantSource struct {
	RoleID          string     `json:"role_id"`
	RoleName        string     `json:"role_name"`
	GrantedByUserID *string    `json:"granted_by_user_id,omitempty"`
	GrantedAt       *time.Time `json:"granted_at,omitempty"`
}

type UserPermissionInfo struct {
	PermissionID          string                  `json:"permission_id"`
	PermissionKey         string                  `json:"permission_key"`
	PermissionDescription *string                 `json:"permission_description,omitempty"`
	GrantedByUserID       *string                 `json:"granted_by_user_id,omitempty"`
	GrantedAt             *time.Time              `json:"granted_at,omitempty"`
	Sources               []PermissionGrantSource `json:"sources,omitempty"`
}

type UserWithRoles struct {
	User  models.User    `json:"user"`
	Roles []UserRoleInfo `json:"roles"`
}

type UserWithPermissions struct {
	User        models.User          `json:"user"`
	Permissions []UserPermissionInfo `json:"permissions"`
}

type UserAuthorizationProfile struct {
	User        models.User          `json:"user"`
	Roles       []UserRoleInfo       `json:"roles"`
	Permissions []UserPermissionInfo `json:"permissions"`
}

type RoleDetails struct {
	Role        Role                 `json:"role"`
	Permissions []UserPermissionInfo `json:"permissions"`
}
