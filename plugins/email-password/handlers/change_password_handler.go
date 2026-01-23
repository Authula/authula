package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
)

type ChangePasswordPayload struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type ChangePasswordHandler struct {
	UseCase *usecases.ChangePasswordUseCase
}

func (h *ChangePasswordHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, ok := models.GetRequestContext(ctx)
		if !ok || reqCtx == nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var payload ChangePasswordPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.ChangePassword(ctx, payload.Token, payload.Password)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "password updated",
		})
	}

}
