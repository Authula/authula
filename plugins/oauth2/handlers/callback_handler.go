package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	"github.com/Authula/authula/util"
)

type CallbackHandler struct {
	UseCase usecases.CallbackUseCase
	Logger  models.Logger
}

func (h *CallbackHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		providerID := strings.TrimSpace(r.PathValue("provider"))
		if providerID == "" {
			WriteError(reqCtx, h.Logger, constants.ErrProviderNotFound)
			reqCtx.Handled = true
			return
		}

		state := util.GetQueryString(r, "state", "")
		if strings.TrimSpace(state) == "" {
			WriteError(reqCtx, h.Logger, constants.ErrInvalidState)
			reqCtx.Handled = true
			return
		}

		bindingCookie, err := r.Cookie(constants.CookieFlowBinding)
		if err != nil || strings.TrimSpace(bindingCookie.Value) == "" {
			WriteError(reqCtx, h.Logger, constants.ErrInvalidState)
			reqCtx.Handled = true
			return
		}

		req := &types.CallbackRequest{
			ProviderID:       providerID,
			Code:             util.GetQueryString(r, "code", ""),
			State:            state,
			Error:            util.GetQueryString(r, "error", ""),
			ErrorDescription: util.GetQueryString(r, "error_description", ""),
			Binding:          bindingCookie.Value,
		}

		if reqCtx.Actor != nil && reqCtx.Actor.Type == models.ActorUser {
			req.ActorID = reqCtx.Actor.ID
			if sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string); ok {
				req.SessionID = sessionID
			}
		}

		userAgent := r.UserAgent()
		result, err := h.UseCase.Callback(ctx, req, &reqCtx.ClientIP, &userAgent)
		if err != nil {
			if redirectErr := new(usecases.RedirectError); errors.As(err, &redirectErr) && redirectErr.RedirectTo != "" {
				h.redirect(reqCtx, redirectErr)
				return
			}
			WriteError(reqCtx, h.Logger, err)
			reqCtx.Handled = true
			return
		}
		if result == nil || result.User == nil || result.Session == nil {
			WriteError(reqCtx, h.Logger, errors.New("oauth callback returned an incomplete result"))
			reqCtx.Handled = true
			return
		}

		reqCtx.SetActorInContext(&models.Actor{
			ID:   result.User.ID,
			Type: models.ActorUser,
		})
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		if !result.ReusedSession && result.SessionToken != "" {
			reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
			reqCtx.Values[models.ContextAuthSuccess.String()] = true
		}

		if result.RedirectTo != "" {
			reqCtx.RedirectURL = result.RedirectTo
			reqCtx.ResponseStatus = http.StatusFound
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, callbackResponse(result))
		if reqCtx.ResponseHeaders == nil {
			reqCtx.ResponseHeaders = make(http.Header)
		}
		reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	}
}

// redirect sends the browser to the stored redirect_to with provider error parameters attached.
func (h *CallbackHandler) redirect(reqCtx *models.RequestContext, redirectErr *usecases.RedirectError) {
	target, err := url.Parse(redirectErr.RedirectTo)
	if err != nil {
		WriteError(reqCtx, h.Logger, constants.ErrAccessDenied)
		reqCtx.Handled = true
		return
	}

	query := target.Query()
	query.Set("error", redirectErr.Code)
	if redirectErr.Description != "" {
		query.Set("error_description", redirectErr.Description)
	}
	target.RawQuery = query.Encode()

	reqCtx.RedirectURL = target.String()
	reqCtx.ResponseStatus = http.StatusFound
	if reqCtx.ResponseHeaders == nil {
		reqCtx.ResponseHeaders = make(http.Header)
	}
	reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	reqCtx.ResponseWriter.Header().Set("Cache-Control", "no-store")
}

func callbackResponse(result *types.CallbackResult) *types.CallbackResponse {
	return &types.CallbackResponse{
		User: result.User,
		Session: &types.CallbackSession{
			ID:        result.Session.ID,
			UserID:    result.Session.UserID,
			ExpiresAt: result.Session.ExpiresAt,
		},
	}
}
