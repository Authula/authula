package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/usecases"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type SignOutRequestPayload struct {
	SessionID  *string `json:"session_id,omitempty"`
	SignOutAll bool    `json:"sign_out_all,omitempty"`
}

type SignOutHandler struct {
	UseCase *usecases.SignOutUseCase
}

func (h *SignOutHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		var payload SignOutRequestPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			// If no body provided, that's okay - we'll use default behavior
			payload = SignOutRequestPayload{}
		}

		result, err := h.UseCase.SignOut(ctx, userID, payload.SessionID, payload.SignOutAll)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": "failed to sign out",
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.Values[models.ContextAuthSignOut.String()] = true

		reqCtx.SetJSONResponse(http.StatusOK, result)
	}
}
