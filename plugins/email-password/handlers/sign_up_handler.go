package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

type SignUpHandler struct {
	Logger                       models.Logger
	Config                       *models.Config
	PluginConfig                 types.EmailPasswordPluginConfig
	SignUpUseCase                usecases.SignUpUseCase
	SendEmailVerificationUseCase usecases.SendEmailVerificationUseCase
}

func (h *SignUpHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var request types.SignUpRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		userAgent := r.UserAgent()
		result, err := h.SignUpUseCase.SignUp(
			ctx,
			request.Name,
			request.Email,
			request.Password,
			request.Image,
			request.Metadata,
			request.CallbackURL,
			&reqCtx.ClientIP,
			&userAgent,
		)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		if h.PluginConfig.RequireEmailVerification && h.PluginConfig.SendEmailOnSignUp {
			go func() {
				detachedCtx := context.WithoutCancel(ctx)
				taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
				defer cancel()

				if err := h.SendEmailVerificationUseCase.Send(taskCtx, request.Email, request.CallbackURL); err != nil {
					h.Logger.Error("failed to send email", "err", err)
				}
			}()
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		if h.PluginConfig.AutoSignIn {
			reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
			reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
			reqCtx.Values[models.ContextAuthSuccess.String()] = true
		}

		reqCtx.SetJSONResponse(http.StatusCreated, types.SignUpResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
