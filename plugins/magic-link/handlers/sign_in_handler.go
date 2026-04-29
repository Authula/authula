package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	"github.com/Authula/authula/plugins/magic-link/usecases"
)

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

		var request types.SignInRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		_, err := h.UseCase.SignIn(ctx, request.Name, request.Email, request.CallbackURL)
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
