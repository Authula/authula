package accesscontrol

import (
	"net/http"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
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
	}
}

func (p *AccessControlPlugin) requireAccessControl(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	requiredPermissions := readStringSliceMetadata(reqCtx, "permissions")
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
			if !ok {
				continue
			}

			trimmed := strings.TrimSpace(str)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return nil
}
