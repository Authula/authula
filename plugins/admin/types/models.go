package types

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// Models

type Role struct {
	bun.BaseModel `bun:"table:admin_roles"`

	ID          string    `json:"id" bun:"column:id,pk"`
	Name        string    `json:"name" bun:"column:name"`
	Description *string   `json:"description" bun:"column:description"`
	IsSystem    bool      `json:"is_system" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type Permission struct {
	bun.BaseModel `bun:"table:admin_permissions"`

	ID          string    `json:"id" bun:"column:id,pk"`
	Key         string    `json:"key" bun:"column:key"`
	Description *string   `json:"description" bun:"column:description"`
	IsSystem    bool      `json:"is_system" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type RolePermission struct {
	bun.BaseModel `bun:"table:admin_role_permissions"`

	RoleID          string    `json:"role_id" bun:"column:role_id,pk"`
	PermissionID    string    `json:"permission_id" bun:"column:permission_id,pk"`
	GrantedByUserID *string   `json:"granted_by_user_id" bun:"column:granted_by_user_id"`
	GrantedAt       time.Time `json:"granted_at" bun:"column:granted_at"`
}

type UserRole struct {
	bun.BaseModel `bun:"table:admin_user_roles"`

	UserID           string     `json:"user_id" bun:"column:user_id,pk"`
	RoleID           string     `json:"role_id" bun:"column:role_id,pk"`
	AssignedByUserID *string    `json:"assigned_by_user_id" bun:"column:assigned_by_user_id"`
	AssignedAt       time.Time  `json:"assigned_at" bun:"column:assigned_at"`
	ExpiresAt        *time.Time `json:"expires_at" bun:"column:expires_at"`
}

type Impersonation struct {
	bun.BaseModel `bun:"table:admin_impersonations"`

	ID                     string     `json:"id" bun:"column:id,pk"`
	ActorUserID            string     `json:"actor_user_id" bun:"column:actor_user_id"`
	TargetUserID           string     `json:"target_user_id" bun:"column:target_user_id"`
	ActorSessionID         *string    `json:"actor_session_id" bun:"column:actor_session_id"`
	ImpersonationSessionID *string    `json:"impersonation_session_id" bun:"column:impersonation_session_id"`
	Reason                 string     `json:"reason" bun:"column:reason"`
	StartedAt              time.Time  `json:"started_at" bun:"column:started_at"`
	ExpiresAt              time.Time  `json:"expires_at" bun:"column:expires_at"`
	EndedAt                *time.Time `json:"ended_at" bun:"column:ended_at"`
	EndedByUserID          *string    `json:"ended_by_user_id" bun:"column:ended_by_user_id"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserState struct {
	bun.BaseModel `bun:"table:admin_user_states"`

	UserID         string     `json:"user_id" bun:"column:user_id,pk"`
	IsBanned       bool       `json:"is_banned" bun:"column:is_banned"`
	BannedAt       *time.Time `json:"banned_at" bun:"column:banned_at"`
	BannedUntil    *time.Time `json:"banned_until" bun:"column:banned_until"`
	BannedReason   *string    `json:"banned_reason" bun:"column:banned_reason"`
	BannedByUserID *string    `json:"banned_by_user_id" bun:"column:banned_by_user_id"`
	CreatedAt      time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminSessionState struct {
	bun.BaseModel `bun:"table:admin_session_states"`

	SessionID              string     `json:"session_id" bun:"column:session_id,pk"`
	RevokedAt              *time.Time `json:"revoked_at" bun:"column:revoked_at"`
	RevokedReason          *string    `json:"revoked_reason" bun:"column:revoked_reason"`
	RevokedByUserID        *string    `json:"revoked_by_user_id" bun:"column:revoked_by_user_id"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id" bun:"column:impersonator_user_id"`
	ImpersonationReason    *string    `json:"impersonation_reason" bun:"column:impersonation_reason"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at" bun:"column:impersonation_expires_at"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

// Types

type GetAllRolesResponse struct {
	Roles []Role `json:"roles"`
}

type CreateRoleRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsSystem    bool    `json:"is_system"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreatePermissionRequest struct {
	Key         string  `json:"key"`
	Description *string `json:"description,omitempty"`
	IsSystem    bool    `json:"is_system"`
}

type UpdatePermissionRequest struct {
	Description *string `json:"description,omitempty"`
}

type AddRolePermissionRequest struct {
	PermissionID string `json:"permission_id"`
}

type ReplaceRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

type AssignUserRoleRequest struct {
	RoleID    string     `json:"role_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ReplaceUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type UserRoleInfo struct {
	RoleID    string     `json:"role_id"`
	RoleName  string     `json:"role_name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type UserPermissionInfo struct {
	PermissionID  string `json:"permission_id"`
	PermissionKey string `json:"permission_key"`
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

type StartImpersonationRequest struct {
	TargetUserID     string `json:"target_user_id"`
	Reason           string `json:"reason"`
	ExpiresInSeconds *int   `json:"expires_in_seconds,omitempty"`
}

type StopImpersonationRequest struct {
	ImpersonationID *string `json:"impersonation_id,omitempty"`
}

type RevokeSessionRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type BanUserRequest struct {
	BannedUntil *time.Time `json:"banned_until,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
}

type RoleDetails struct {
	Role        Role                 `json:"role"`
	Permissions []UserPermissionInfo `json:"permissions"`
}

type AdminUserSession struct {
	Session models.Session     `json:"session"`
	State   *AdminSessionState `json:"state,omitempty"`
}

type UpsertUserStateRequest struct {
	IsBanned     bool       `json:"is_banned"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}

type UpsertSessionStateRequest struct {
	Revoke                 bool       `json:"revoke"`
	RevokedReason          *string    `json:"revoked_reason,omitempty"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty"`
}

type CreateUserRequest struct {
	Name          string          `json:"name"`
	Email         string          `json:"email"`
	EmailVerified *bool           `json:"email_verified,omitempty"`
	Image         *string         `json:"image,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type UpdateUserRequest struct {
	Name          *string         `json:"name,omitempty"`
	Email         *string         `json:"email,omitempty"`
	EmailVerified *bool           `json:"email_verified,omitempty"`
	Image         *string         `json:"image,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type StartImpersonationResult struct {
	Impersonation *Impersonation `json:"impersonation"`
	SessionID     *string        `json:"session_id,omitempty"`
	SessionToken  *string        `json:"session_token,omitempty"`
}

type UsersPage struct {
	Users      []models.User `json:"users"`
	NextCursor *string       `json:"next_cursor,omitempty"`
}
