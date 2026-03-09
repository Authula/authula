package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type GetAllImpersonationsHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewGetAllImpersonationsHandler(useCase usecases.ImpersonationUseCase) *GetAllImpersonationsHandler {
	return &GetAllImpersonationsHandler{useCase: useCase}
}

func (h *GetAllImpersonationsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		rows, err := h.useCase.GetAllImpersonations(r.Context())
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, rows)
	}
}

type GetImpersonationByIDHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewGetImpersonationByIDHandler(useCase usecases.ImpersonationUseCase) *GetImpersonationByIDHandler {
	return &GetImpersonationByIDHandler{useCase: useCase}
}

func (h *GetImpersonationByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		impersonationID := r.PathValue("impersonation_id")

		impersonation, err := h.useCase.GetImpersonationByID(r.Context(), impersonationID)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetImpersonationByIDResponse{Impersonation: impersonation})
	}
}

type StartImpersonationHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewStartImpersonationHandler(useCase usecases.ImpersonationUseCase) *StartImpersonationHandler {
	return &StartImpersonationHandler{useCase: useCase}
}

func (h *StartImpersonationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		impersonatorUserID := getUserID(reqCtx)

		if impersonatorUserID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		var payload types.StartImpersonationRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.useCase.StartImpersonation(r.Context(), *impersonatorUserID, getSessionID(reqCtx), &reqCtx.ClientIP, &userAgent, payload)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		if result == nil || result.Impersonation == nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to start impersonation"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetUserIDInContext(result.Impersonation.TargetUserID)
		if result.SessionID != nil && *result.SessionID != "" {
			reqCtx.Values[models.ContextSessionID.String()] = *result.SessionID
		}
		if result.SessionToken != nil && *result.SessionToken != "" {
			reqCtx.Values[models.ContextSessionToken.String()] = *result.SessionToken
			reqCtx.Values[models.ContextAuthSuccess.String()] = true
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.StartImpersonationResponse{
			Impersonation: result.Impersonation,
		})
	}
}

type StopImpersonationHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewStopImpersonationHandler(useCase usecases.ImpersonationUseCase) *StopImpersonationHandler {
	return &StopImpersonationHandler{useCase: useCase}
}

func (h *StopImpersonationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		impersonatedUserID := getUserID(reqCtx)
		impersonatedSessionID := getSessionID(reqCtx)

		if impersonatedUserID == nil || impersonatedSessionID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		impersonationID := r.PathValue("impersonation_id")
		if err := h.useCase.StopImpersonation(r.Context(), *impersonatedUserID, *impersonatedSessionID, types.StopImpersonationRequest{ImpersonationID: &impersonationID}); err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		reqCtx.Values[models.ContextAuthSignOut.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, &types.StopImpersonationResponse{Message: "Impersonation stopped"})
	}
}

func getUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func getSessionID(reqCtx *models.RequestContext) *string {
	value, ok := reqCtx.Values[models.ContextSessionID.String()]
	if !ok || value == nil {
		return nil
	}

	sessionID, ok := value.(string)
	if !ok || sessionID == "" {
		return nil
	}

	return &sessionID
}

func respondImpersonationError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapImpersonationErrorStatus(err), map[string]any{"message": mapAdminHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapImpersonationErrorStatus(err error) int {
	return mapAdminHttpErrorStatus(err)
}
