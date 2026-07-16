package types

import (
	"time"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

type RoleID struct {
	RoleID string `path:"role_id"`
}

type RoleName struct {
	RoleName string `path:"role_name"`
}

type PermissionID struct {
	PermissionID string `path:"permission_id"`
}

type PermissionKey struct {
	PermissionKey string `path:"permission_key"`
}

type UserID struct {
	UserID string `path:"user_id"`
}

type RolePermissionID struct {
	RoleID       string `path:"role_id"`
	PermissionID string `path:"permission_id"`
}

type UserRoleID struct {
	UserID string `path:"user_id"`
	RoleID string `path:"role_id"`
}

type CreateRoleRequest struct {
	Name        string  `json:"name" required:"true" nullable:"false"`
	Description *string `json:"description,omitempty" nullable:"true"`
	Weight      *int    `json:"weight,omitempty" nullable:"true"`
	IsSystem    bool    `json:"is_system" required:"true" nullable:"false"`
}

func (req *CreateRoleRequest) Validate() error {
	if req.Name == "" {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type CreateRoleResponse struct {
	Role *Role `json:"role" required:"true" nullable:"false"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty" nullable:"true"`
	Description *string `json:"description,omitempty" nullable:"true"`
	Weight      *int    `json:"weight,omitempty" nullable:"true"`
}

func (req *UpdateRoleRequest) Validate() error {
	if req.Name == nil && req.Description == nil && req.Weight == nil {
		return accesscontrolconstants.ErrAtleastOneFieldRequiredToUpdateResource
	}
	return nil
}

type UpdateRoleResponse struct {
	Role *Role `json:"role" required:"true" nullable:"false"`
}

type DeleteRoleResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type CreatePermissionRequest struct {
	Key         string  `json:"key" required:"true" nullable:"false"`
	Description *string `json:"description,omitempty" nullable:"true"`
	IsSystem    bool    `json:"is_system" nullable:"false"`
}

func (req *CreatePermissionRequest) Validate() error {
	if req.Key == "" {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type CreatePermissionResponse struct {
	Permission *Permission `json:"permission" required:"true" nullable:"false"`
}

type UpdatePermissionRequest struct {
	Description *string `json:"description,omitempty" nullable:"true"`
}

type UpdatePermissionResponse struct {
	Permission *Permission `json:"permission" required:"true" nullable:"false"`
}

type DeletePermissionResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type AddRolePermissionRequest struct {
	PermissionID string `json:"permission_id" required:"true" nullable:"false"`
}

func (req *AddRolePermissionRequest) Validate() error {
	if req.PermissionID == "" {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type AddRolePermissionResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type ReplaceRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids" required:"true" nullable:"false"`
}

func (req *ReplaceRolePermissionsRequest) Validate() error {
	if len(req.PermissionIDs) == 0 {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type ReplaceRolePermissionResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type RemoveRolePermissionResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type AssignUserRoleRequest struct {
	RoleID    string     `json:"role_id" required:"true" nullable:"false"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" nullable:"true"`
}

func (req *AssignUserRoleRequest) Validate() error {
	if req.RoleID == "" {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type ReplaceUserRolesRequest struct {
	RoleIDs []string `json:"role_ids" required:"true" nullable:"false"`
}

func (req *ReplaceUserRolesRequest) Validate() error {
	if len(req.RoleIDs) == 0 {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type ReplaceUserRolesResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type AssignUserRoleResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type RemoveUserRoleResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type CheckUserPermissionsRequest struct {
	PermissionKeys []string `json:"permission_keys" required:"true" nullable:"false"`
}

func (req *CheckUserPermissionsRequest) Validate() error {
	if len(req.PermissionKeys) == 0 {
		return coreerrors.ErrUnprocessableEntity
	}
	return nil
}

type CheckUserPermissionsResponse struct {
	HasPermissions bool `json:"has_permissions" required:"true" nullable:"false"`
}

type GetUserPermissionsResponse struct {
	Permissions []UserPermissionInfo `json:"permissions" required:"true" nullable:"false"`
}

type UserRoleInfo struct {
	RoleID           string     `json:"role_id" required:"true" nullable:"false"`
	RoleName         string     `json:"role_name" required:"true" nullable:"false"`
	RoleDescription  *string    `json:"role_description,omitempty" nullable:"true"`
	RoleWeight       int        `json:"role_weight" required:"true" nullable:"false"`
	AssignedByUserID *string    `json:"assigned_by_user_id,omitempty" nullable:"true"`
	AssignedAt       *time.Time `json:"assigned_at,omitempty" nullable:"true"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty" nullable:"true"`
}

type PermissionGrantSource struct {
	RoleID          string     `json:"role_id" required:"true" nullable:"false"`
	RoleName        string     `json:"role_name" required:"true" nullable:"false"`
	GrantedByUserID *string    `json:"granted_by_user_id,omitempty" nullable:"true"`
	GrantedAt       *time.Time `json:"granted_at,omitempty" nullable:"true"`
}

type UserPermissionInfo struct {
	PermissionID          string                  `json:"permission_id" required:"true" nullable:"false"`
	PermissionKey         string                  `json:"permission_key" required:"true" nullable:"false"`
	PermissionDescription *string                 `json:"permission_description,omitempty" nullable:"true"`
	GrantedByUserID       *string                 `json:"granted_by_user_id,omitempty" nullable:"true"`
	GrantedAt             *time.Time              `json:"granted_at,omitempty" nullable:"true"`
	Sources               []PermissionGrantSource `json:"sources,omitempty" nullable:"true"`
}

type UserWithPermissions struct {
	User        models.User          `json:"user" required:"true" nullable:"false"`
	Permissions []UserPermissionInfo `json:"permissions" required:"true" nullable:"false"`
}

type RoleDetails struct {
	Role        Role                 `json:"role" required:"true" nullable:"false"`
	Permissions []UserPermissionInfo `json:"permissions" required:"true" nullable:"false"`
}
