package types

import (
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

type CreateRoleRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Weight      *int    `json:"weight,omitempty"`
	IsSystem    bool    `json:"is_system"`
}

func (req *CreateRoleRequest) Validate() error {
	if req.Name == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type CreateRoleResponse struct {
	Role *Role `json:"role"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Weight      *int    `json:"weight,omitempty"`
}

func (req *UpdateRoleRequest) Validate() error {
	if req.Name == nil && req.Description == nil && req.Weight == nil {
		return accesscontrolconstants.ErrAtleastOneFieldRequiredToUpdateResource
	}
	return nil
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

func (req *CreatePermissionRequest) Validate() error {
	if req.Key == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
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

func (req *AddRolePermissionRequest) Validate() error {
	if req.PermissionID == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type AddRolePermissionResponse struct {
	Message string `json:"message"`
}

type ReplaceRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

func (req *ReplaceRolePermissionsRequest) Validate() error {
	if len(req.PermissionIDs) == 0 {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
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

func (req *AssignUserRoleRequest) Validate() error {
	if req.RoleID == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type ReplaceUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

func (req *ReplaceUserRolesRequest) Validate() error {
	if len(req.RoleIDs) == 0 {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
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

type CheckUserPermissionsRequest struct {
	PermissionKeys []string `json:"permission_keys"`
}

func (req *CheckUserPermissionsRequest) Validate() error {
	if len(req.PermissionKeys) == 0 {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type CheckUserPermissionsResponse struct {
	HasPermissions bool `json:"has_permissions"`
}

type GetUserPermissionsResponse struct {
	Permissions []UserPermissionInfo `json:"permissions"`
}

type UserRoleInfo struct {
	RoleID           string     `json:"role_id"`
	RoleName         string     `json:"role_name"`
	RoleDescription  *string    `json:"role_description,omitempty"`
	RoleWeight       int        `json:"role_weight"`
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

type UserWithPermissions struct {
	User        models.User          `json:"user"`
	Permissions []UserPermissionInfo `json:"permissions"`
}

type RoleDetails struct {
	Role        Role                 `json:"role"`
	Permissions []UserPermissionInfo `json:"permissions"`
}
