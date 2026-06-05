package handlers

import "github.com/Authula/authula/models"

func rolePermissionActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" {
		return nil
	}
	return &reqCtx.Actor.ID
}

func respondRolePermissionError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapRolePermissionErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapRolePermissionErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}

func userActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" {
		return nil
	}
	return &reqCtx.Actor.ID
}

func respondUserHandlerError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapUserHandlerErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapUserHandlerErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}
