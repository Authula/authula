package accesscontrol

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
)

type AccessControlHookID string

const (
	HookIDAccessControlEnforce AccessControlHookID = "access_control.enforce"
)

func (id AccessControlHookID) String() string {
	return string(id)
}

func (p *AccessControlPlugin) Hooks() []models.Hook {
	return []models.Hook{
		{
			Stage:    models.HookBefore,
			PluginID: HookIDAccessControlEnforce.String(),
			Handler:  p.requireAccessControl,
			Order:    20,
		},
		{
			Stage:   models.HookAfter,
			Handler: p.assignRoleFromContextHook,
			Order:   20,
		},
	}
}

func (p *AccessControlPlugin) assignRoleFromContextHook(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	rawValue, ok := reqCtx.Values[models.ContextAccessControlAssignRole.String()]
	if !ok || rawValue == nil {
		return nil
	}

	assignCtx, ok := accessControlAssignRoleContext(rawValue)
	if !ok || assignCtx.UserID == "" || assignCtx.RoleName == "" {
		return nil
	}

	targetRole, err := p.Api.GetRoleByName(ctx, assignCtx.RoleName)
	if err != nil {
		p.logAssignRoleHookError("failed to resolve role", assignCtx, err)
		return nil
	}

	userRoles, err := p.Api.GetUserRoles(ctx, assignCtx.UserID)
	if err != nil {
		p.logAssignRoleHookError("failed to load user roles", assignCtx, err)
		return nil
	}

	for _, userRole := range userRoles {
		if userRole.RoleName == assignCtx.RoleName {
			return nil
		}
	}

	if err := p.Api.AssignRoleToUser(ctx, assignCtx.UserID, types.AssignUserRoleRequest{RoleID: targetRole.ID}, assignCtx.AssignerUserID); err != nil {
		p.logAssignRoleHookError("failed to assign role", assignCtx, err)
	}

	return nil
}

func (p *AccessControlPlugin) logAssignRoleHookError(message string, assignCtx models.AccessControlAssignRoleContext, err error) {
	assignerUserID := "not provided"
	if assignCtx.AssignerUserID != nil && *assignCtx.AssignerUserID != "" {
		assignerUserID = *assignCtx.AssignerUserID
	}
	p.logger.Error(
		message,
		"user_id", assignCtx.UserID,
		"role_name", assignCtx.RoleName,
		"assigned_by_user_id", assignerUserID,
		"error", err,
	)
}

func accessControlAssignRoleContext(value any) (models.AccessControlAssignRoleContext, bool) {
	switch typed := value.(type) {
	case models.AccessControlAssignRoleContext:
		return typed, true
	case *models.AccessControlAssignRoleContext:
		if typed == nil {
			return models.AccessControlAssignRoleContext{}, false
		}
		return *typed, true
	default:
		return models.AccessControlAssignRoleContext{}, false
	}
}

func (p *AccessControlPlugin) requireAccessControl(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	requiredPermissions := util.ReadStringSliceMetadata(reqCtx, "permissions")
	if len(requiredPermissions) == 0 {
		// Opt-in mode: if no permissions metadata is present, skip access control enforcement.
		return nil
	}

	allowed, err := p.Api.HasPermissions(ctx, *reqCtx.UserID, requiredPermissions)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return nil
	}

	if !allowed {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "Forbidden"})
		reqCtx.Handled = true
		return nil
	}

	return nil
}
