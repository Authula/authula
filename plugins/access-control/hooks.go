package accesscontrol

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
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
