package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type GetUserEffectivePermissionsHandler struct {
	useCase *usecases.UserAccessUseCase
}

func NewGetUserEffectivePermissionsHandler(useCase *usecases.UserAccessUseCase) *GetUserEffectivePermissionsHandler {
	return &GetUserEffectivePermissionsHandler{
		useCase: useCase,
	}
}

func (h *GetUserEffectivePermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		permissions, err := h.useCase.GetUserEffectivePermissions(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetUserEffectivePermissionsResponse{Permissions: permissions})
	}
}

type GetUserAuthorizationProfileHandler struct {
	useCase *usecases.UserAccessUseCase
}

func NewGetUserAuthorizationProfileHandler(useCase *usecases.UserAccessUseCase) *GetUserAuthorizationProfileHandler {
	return &GetUserAuthorizationProfileHandler{useCase: useCase}
}

func (h *GetUserAuthorizationProfileHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		profile, err := h.useCase.GetUserAuthorizationProfile(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, profile)
	}
}
