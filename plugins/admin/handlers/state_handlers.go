package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type GetUserStateHandler struct {
	useCase usecases.StateUseCase
}

func NewGetUserStateHandler(useCase usecases.StateUseCase) *GetUserStateHandler {
	return &GetUserStateHandler{useCase: useCase}
}

func (h *GetUserStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		state, err := h.useCase.GetUserState(r.Context(), userID)
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}
		if state == nil {
			reqCtx.SetJSONResponse(http.StatusNotFound, map[string]any{"message": "user state not found"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetUserStateResponse{State: state})
	}
}

type UpsertUserStateHandler struct {
	useCase usecases.StateUseCase
}

func NewUpsertUserStateHandler(useCase usecases.StateUseCase) *UpsertUserStateHandler {
	return &UpsertUserStateHandler{useCase: useCase}
}

func (h *UpsertUserStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		var payload types.UpsertUserStateRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		state, err := h.useCase.UpsertUserState(r.Context(), userID, payload, stateActorUserID(reqCtx))
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UpsertUserStateResponse{State: state})
	}
}

type DeleteUserStateHandler struct {
	useCase usecases.StateUseCase
}

func NewDeleteUserStateHandler(useCase usecases.StateUseCase) *DeleteUserStateHandler {
	return &DeleteUserStateHandler{useCase: useCase}
}

func (h *DeleteUserStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		if err := h.useCase.DeleteUserState(r.Context(), userID); err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeleteUserStateResponse{Message: "user state deleted"})
	}
}

type GetBannedUserStatesHandler struct {
	useCase usecases.StateUseCase
}

func NewGetBannedUserStatesHandler(useCase usecases.StateUseCase) *GetBannedUserStatesHandler {
	return &GetBannedUserStatesHandler{useCase: useCase}
}

func (h *GetBannedUserStatesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		rows, err := h.useCase.GetBannedUserStates(r.Context())
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, rows)
	}
}

type BanUserHandler struct {
	useCase usecases.StateUseCase
}

func NewBanUserHandler(useCase usecases.StateUseCase) *BanUserHandler {
	return &BanUserHandler{useCase: useCase}
}

func (h *BanUserHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		var payload types.BanUserRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		state, err := h.useCase.BanUser(r.Context(), userID, payload, stateActorUserID(reqCtx))
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.BanUserResponse{State: state})
	}
}

type UnbanUserHandler struct {
	useCase usecases.StateUseCase
}

func NewUnbanUserHandler(useCase usecases.StateUseCase) *UnbanUserHandler {
	return &UnbanUserHandler{useCase: useCase}
}

func (h *UnbanUserHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		state, err := h.useCase.UnbanUser(r.Context(), userID)
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UnbanUserResponse{State: state})
	}
}

type GetSessionStateHandler struct {
	useCase usecases.StateUseCase
}

func NewGetSessionStateHandler(useCase usecases.StateUseCase) *GetSessionStateHandler {
	return &GetSessionStateHandler{useCase: useCase}
}

func (h *GetSessionStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		sessionID := r.PathValue("session_id")

		state, err := h.useCase.GetSessionState(r.Context(), sessionID)
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}
		if state == nil {
			reqCtx.SetJSONResponse(http.StatusNotFound, map[string]any{"message": "session state not found"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetSessionStateResponse{State: state})
	}
}

type UpsertSessionStateHandler struct {
	useCase usecases.StateUseCase
}

func NewUpsertSessionStateHandler(useCase usecases.StateUseCase) *UpsertSessionStateHandler {
	return &UpsertSessionStateHandler{useCase: useCase}
}

func (h *UpsertSessionStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		sessionID := r.PathValue("session_id")

		var payload types.UpsertSessionStateRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		state, err := h.useCase.UpsertSessionState(r.Context(), sessionID, payload, stateActorUserID(reqCtx))
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UpsertSessionStateResponse{State: state})
	}
}

type DeleteSessionStateHandler struct {
	useCase usecases.StateUseCase
}

func NewDeleteSessionStateHandler(useCase usecases.StateUseCase) *DeleteSessionStateHandler {
	return &DeleteSessionStateHandler{useCase: useCase}
}

func (h *DeleteSessionStateHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		sessionID := r.PathValue("session_id")

		if err := h.useCase.DeleteSessionState(r.Context(), sessionID); err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeleteSessionStateResponse{Message: "session state deleted"})
	}
}

type GetRevokedSessionStatesHandler struct {
	useCase usecases.StateUseCase
}

func NewGetRevokedSessionStatesHandler(useCase usecases.StateUseCase) *GetRevokedSessionStatesHandler {
	return &GetRevokedSessionStatesHandler{useCase: useCase}
}

func (h *GetRevokedSessionStatesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		rows, err := h.useCase.GetRevokedSessionStates(r.Context())
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, rows)
	}
}

type GetUserAdminSessionsHandler struct {
	useCase usecases.StateUseCase
}

func NewGetUserAdminSessionsHandler(useCase usecases.StateUseCase) *GetUserAdminSessionsHandler {
	return &GetUserAdminSessionsHandler{useCase: useCase}
}

func (h *GetUserAdminSessionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		rows, err := h.useCase.GetUserAdminSessions(r.Context(), userID)
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, rows)
	}
}

type RevokeSessionHandler struct {
	useCase usecases.StateUseCase
}

func NewRevokeSessionHandler(useCase usecases.StateUseCase) *RevokeSessionHandler {
	return &RevokeSessionHandler{useCase: useCase}
}

func (h *RevokeSessionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		sessionID := r.PathValue("session_id")

		var payload types.RevokeSessionRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		state, err := h.useCase.RevokeSession(r.Context(), sessionID, payload.Reason, stateActorUserID(reqCtx))
		if err != nil {
			respondStateError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.RevokeSessionResponse{State: state})
	}
}

func stateActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx == nil || reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func respondStateError(reqCtx *models.RequestContext, err error) {
	if reqCtx == nil {
		return
	}

	reqCtx.SetJSONResponse(mapStateErrorStatus(err), map[string]any{"message": mapAdminHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapStateErrorStatus(err error) int {
	return mapAdminHttpErrorStatus(err)
}
