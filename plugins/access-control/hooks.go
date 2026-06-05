package accesscontrol

import (
	"net/http"
	"slices"

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

func (p *AccessControlPlugin) requireAccessControl(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	requiredPermissions := util.ReadStringSliceMetadata(reqCtx, "permissions")
	if len(requiredPermissions) == 0 {
		return nil // Opt-in mode: skip if no permission flags are set on the route
	}

	var allowed bool
	var err error

	switch reqCtx.Actor.Type {
	case models.ActorMachine:
		{
			// For machine actors, we check if they have the required permissions directly in their scopes.
			allowed = hasAllScopes(reqCtx.Actor.Scopes, requiredPermissions)
		}
	case models.ActorUser:
		{
			// For user actors, we need to check their assigned roles and permissions from the tables in the DB.
			allowed, err = p.Api.HasPermissions(ctx, reqCtx.Actor.ID, requiredPermissions)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
				reqCtx.Handled = true
				return nil
			}
		}
	}

	if !allowed {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "Forbidden"})
		reqCtx.Handled = true
		return nil
	}

	return nil
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

func hasAllScopes(assignedScopes []string, requiredPermissions []string) bool {
	for _, req := range requiredPermissions {
		found := slices.Contains(assignedScopes, req)
		if !found {
			return false
		}
	}
	return true
}
