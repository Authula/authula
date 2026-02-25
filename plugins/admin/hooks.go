package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type AdminHookID string

const (
	HookIDAdminRBAC AdminHookID = "admin.rbac"
)

func (id AdminHookID) String() string {
	return string(id)
}

func (p *AdminPlugin) Hooks() []models.Hook {
	if p.Api == nil {
		p.logger.Warn("Api reference is not set for Admin plugin, returning empty hooks.")
		return []models.Hook{}
	}

	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Handler: p.enforceAdminState,
			Order:   15,
		},
		{
			Stage:    models.HookBefore,
			PluginID: HookIDAdminRBAC.String(),
			Handler:  p.requireRBAC,
			Order:    20,
		},
	}
}

func (p *AdminPlugin) enforceAdminState(reqCtx *models.RequestContext) error {
	if reqCtx == nil || reqCtx.Request == nil {
		return nil
	}

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}

	ctx := reqCtx.Request.Context()

	state, err := p.Api.GetUserState(ctx, *reqCtx.UserID)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate user state"})
		reqCtx.Handled = true
		return nil
	}

	if state != nil && state.IsBanned {
		if state.BannedUntil == nil || state.BannedUntil.After(time.Now().UTC()) {
			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "account is banned"})
			reqCtx.Handled = true
			return nil
		}
	}

	rawSessionID, hasSessionID := reqCtx.Values[models.ContextSessionID.String()]
	if !hasSessionID || rawSessionID == nil {
		return nil
	}

	sessionID, ok := rawSessionID.(string)
	if !ok || sessionID == "" {
		return nil
	}

	sessionState, err := p.Api.GetSessionState(ctx, sessionID)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate session state"})
		reqCtx.Handled = true
		return nil
	}

	if sessionState != nil && sessionState.RevokedAt != nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "session is revoked"})
		reqCtx.Handled = true
		return nil
	}

	return nil
}

func (p *AdminPlugin) requireRBAC(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	requiredPermissions := readStringSliceMetadata(reqCtx, "permissions")
	if len(requiredPermissions) == 0 {
		reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "Forbidden"})
		reqCtx.Handled = true
		return nil
	}

	allowed, err := p.Api.HasPermissions(ctx, *reqCtx.UserID, requiredPermissions)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate permissions"})
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

func readStringSliceMetadata(reqCtx *models.RequestContext, key string) []string {
	if reqCtx == nil || reqCtx.Route == nil || reqCtx.Route.Metadata == nil {
		return nil
	}

	raw, ok := reqCtx.Route.Metadata[key]
	if !ok || raw == nil {
		return nil
	}

	if values, ok := raw.([]string); ok {
		result := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	if valuesAny, ok := raw.([]any); ok {
		result := make([]string, 0, len(valuesAny))
		for _, value := range valuesAny {
			str, ok := value.(string)
			if ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}

	return nil
}
