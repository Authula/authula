package types

import (
	"time"

	"github.com/uptrace/bun"
)

type Role struct {
	bun.BaseModel `bun:"table:access_control_roles"`

	ID          string    `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	Name        string    `json:"name" required:"true" nullable:"false" bun:"column:name"`
	Description *string   `json:"description" nullable:"true" bun:"column:description"`
	Weight      int       `json:"weight" required:"true" nullable:"false" bun:"column:weight"`
	IsSystem    bool      `json:"is_system" required:"true" nullable:"false" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type Permission struct {
	bun.BaseModel `bun:"table:access_control_permissions"`

	ID          string    `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	Key         string    `json:"key" required:"true" nullable:"false" bun:"column:key"`
	Description *string   `json:"description" nullable:"true" bun:"column:description"`
	IsSystem    bool      `json:"is_system" required:"true" nullable:"false" bun:"column:is_system"`
	CreatedAt   time.Time `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type RolePermission struct {
	bun.BaseModel `bun:"table:access_control_role_permissions"`

	RoleID          string    `json:"role_id" required:"true" nullable:"false" bun:"column:role_id,pk"`
	PermissionID    string    `json:"permission_id" required:"true" nullable:"false" bun:"column:permission_id,pk"`
	GrantedByUserID *string   `json:"granted_by_user_id" nullable:"true" bun:"column:granted_by_user_id"`
	GrantedAt       time.Time `json:"granted_at" required:"true" nullable:"false" bun:"column:granted_at"`
}

type UserRole struct {
	bun.BaseModel `bun:"table:access_control_user_roles"`

	UserID           string     `json:"user_id" required:"true" nullable:"false" bun:"column:user_id,pk"`
	RoleID           string     `json:"role_id" required:"true" nullable:"false" bun:"column:role_id,pk"`
	AssignedByUserID *string    `json:"assigned_by_user_id" nullable:"true" bun:"column:assigned_by_user_id"`
	AssignedAt       time.Time  `json:"assigned_at" required:"true" nullable:"false" bun:"column:assigned_at"`
	ExpiresAt        *time.Time `json:"expires_at" nullable:"true" bun:"column:expires_at"`
}
