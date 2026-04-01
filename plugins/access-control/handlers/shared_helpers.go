package handlers

import "github.com/Authula/authula/models"

func rolePermissionActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func respondRolePermissionError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapRolePermissionErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapRolePermissionErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}

func userActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func respondUserHandlerError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapUserHandlerErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapUserHandlerErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}
