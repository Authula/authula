package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type SendEmailVerificationHandler struct {
	UseCase usecases.SendEmailVerificationUseCase
}

func (h *SendEmailVerificationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID == nil || *reqCtx.UserID == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
			reqCtx.Handled = true
			return
		}

		var request types.SendEmailVerificationRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid json body"})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		err := h.UseCase.Send(ctx, *reqCtx.UserID, request.CallbackURL)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "If an account exists with this email, a verification link has been sent."})
	}
}
