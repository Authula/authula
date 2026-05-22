package handlers

import "github.com/Authula/authula/models"

func rolePermissionActorUserID(reqCtx *models.RequestContext) *string {
	principal, ok := models.GetActorFromContext(reqCtx.Request.Context())
	if !ok || principal == nil || principal.ID == "" {
		return nil
	}
	return &principal.ID
}

func respondRolePermissionError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapRolePermissionErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapRolePermissionErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}

func userActorUserID(reqCtx *models.RequestContext) *string {
	principal, ok := models.GetActorFromContext(reqCtx.Request.Context())
	if !ok || principal == nil || principal.ID == "" {
		return nil
	}
	return &principal.ID
}

func respondUserHandlerError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapUserHandlerErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapUserHandlerErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}
