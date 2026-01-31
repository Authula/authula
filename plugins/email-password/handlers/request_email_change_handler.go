package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type RequestEmailChangePayload struct {
	NewEmail    string `json:"new_email"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type RequestEmailChangeHandler struct {
	UseCase *usecases.RequestEmailChangeUseCase
}

func (h *RequestEmailChangeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
			reqCtx.Handled = true
			return
		}

		var payload RequestEmailChangePayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.RequestChange(ctx, *reqCtx.UserID, payload.NewEmail, &payload.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "verification sent to new email",
		})
	}
}
