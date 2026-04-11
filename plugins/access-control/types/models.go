package types

import (
	"time"

	"github.com/uptrace/bun"
)

type Role struct {
	bun.BaseModel `bun:"table:access_control_roles"`

	ID          string    `json:"id" bun:"column:id,pk"`
	Name        string    `json:"name" bun:"column:name"`
	Description *string   `json:"description" bun:"column:description"`
	Weight      int       `json:"weight" bun:"column:weight"`
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
