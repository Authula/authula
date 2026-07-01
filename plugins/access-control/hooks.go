package accesscontrol

import (
	"net/http"
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
)

func (p *AccessControlPlugin) Hooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Handler: p.requireAccessControl,
			Order:   20,
		},
		{
			Stage:   models.HookAfter,
			Handler: p.assignRoleFromContextHook,
			Order:   20,
		},
	}
}

func (p *AccessControlPlugin) requireAccessControl(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" {
		return nil
	}

	if err := p.hydrateActorScopes(reqCtx); err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return nil
	}

	requiredPermissions := util.ReadStringSliceMetadata(reqCtx, "permissions")
	if len(requiredPermissions) == 0 {
		return nil
	}

	if !hasAllScopes(reqCtx.Actor.Scopes, requiredPermissions) {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "Forbidden"})
		reqCtx.Handled = true
	}

	return nil
}

func (p *AccessControlPlugin) hydrateActorScopes(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	switch reqCtx.Actor.Type {
	case models.ActorUser:
		userPermissions, err := p.Api.GetSelfUserPermissions(ctx, reqCtx.Actor, reqCtx.Actor.ID)
		if err != nil {
			return err
		}
		activeUserScopes := extractPermissionKeys(userPermissions)

		if len(reqCtx.Actor.Scopes) == 0 {
			// This is a standard user session -> grant full active scopes
			reqCtx.Actor.Scopes = activeUserScopes
		} else {
			// This is a Personal API Key -> Validate against stale/revoked permissions
			for _, apiKeyScope := range reqCtx.Actor.Scopes {
				if !hasScope(activeUserScopes, apiKeyScope) {
					// Strict Invalidation: The user lost this permission, kill the flight
					return internalerrors.ErrForbidden
				}
			}
		}
	case models.ActorMachine:
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

	targetRole, err := p.Api.GetRoleByName(ctx, reqCtx.Actor, assignCtx.RoleName)
	if err != nil {
		p.logAssignRoleHookError("failed to resolve role", assignCtx, err)
		return nil
	}

	userRoles, err := p.Api.GetUserRoles(ctx, reqCtx.Actor, assignCtx.UserID)
	if err != nil {
		p.logAssignRoleHookError("failed to load user roles", assignCtx, err)
		return nil
	}

	for _, userRole := range userRoles {
		if userRole.RoleName == assignCtx.RoleName {
			return nil
		}
	}

	if err := p.Api.AssignRoleToUser(ctx, reqCtx.Actor, assignCtx.UserID, types.AssignUserRoleRequest{RoleID: targetRole.ID}, assignCtx.AssignerUserID); err != nil {
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

func hasScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == "*" || scope == target {
			return true
		}
		if before, ok := strings.CutSuffix(scope, "*"); ok {
			if strings.HasPrefix(target, before) {
				return true
			}
		}
	}
	return false
}

func hasAllScopes(assignedScopes []string, requiredPermissions []string) bool {
	for _, req := range requiredPermissions {
		if !hasScope(assignedScopes, req) {
			return false
		}
	}
	return true
}

func extractPermissionKeys(permissions []types.UserPermissionInfo) []string {
	keys := make([]string, len(permissions))
	for i, p := range permissions {
		keys[i] = p.PermissionKey
	}
	return keys
}
