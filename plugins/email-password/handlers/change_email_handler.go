package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type ChangeEmailPayload struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

type ChangeEmailHandler struct {
	UseCase *usecases.ChangeEmailUseCase
}

func (h *ChangeEmailHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
			reqCtx.Handled = true
			return
		}

		var payload ChangeEmailPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.ChangeEmail(ctx, *reqCtx.UserID, payload.Token, payload.Email)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "email updated",
		})
	}
}
