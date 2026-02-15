package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/usecases"
)

type SignInPayload struct {
	Email       string  `json:"email"`
	Name        *string `json:"name,omitempty"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type SignInHandler struct {
	UseCase usecases.SignInUseCase
}

func (h *SignInHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID != nil && *reqCtx.UserID != "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "you're already authenticated.",
			})
			reqCtx.Handled = true
			return
		}

		var payload SignInPayload
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		_, err := h.UseCase.SignIn(ctx, payload.Name, payload.Email, payload.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.SignInResponse{
			Message: "if an account exists for this email, a magic link has been sent.",
		})
	}
}
