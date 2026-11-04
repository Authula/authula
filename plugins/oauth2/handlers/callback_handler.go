package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
)

// CallbackUseCase is the handler-facing contract for completing an OAuth2 flow.
type CallbackUseCase interface {
	Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error)
}

type CallbackHandler struct {
	UseCase CallbackUseCase
	Cookie  CookieOptions
	Logger  models.Logger
}

func (h *CallbackHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, ok := models.GetRequestContext(ctx)
		if !ok || reqCtx == nil {
			http.Error(w, "request context missing", http.StatusInternalServerError)
			return
		}

		// OAuth2 flow cookies are single-use; clear them on every terminal path.
		defer h.Cookie.clearFlowCookies(reqCtx.ResponseWriter)

		providerID := strings.TrimSpace(r.PathValue("provider"))
		if providerID == "" {
			WriteError(reqCtx, h.Logger, constants.ErrProviderNotFound)
			reqCtx.Handled = true
			return
		}

		stateCookie, err := r.Cookie(constants.CookieState)
		if err != nil {
			WriteError(reqCtx, h.Logger, constants.ErrInvalidState)
			reqCtx.Handled = true
			return
		}

		req := &types.CallbackRequest{
			ProviderID:  providerID,
			Code:        r.URL.Query().Get("code"),
			State:       r.URL.Query().Get("state"),
			Error:       r.URL.Query().Get("error"),
			StateCookie: stateCookie.Value,
		}
		if cookie, err := r.Cookie(constants.CookieRedirectTo); err == nil {
			req.RedirectCookie = cookie.Value
		}
		if cookie, err := r.Cookie(constants.CookieVerifier); err == nil {
			req.VerifierCookie = cookie.Value
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
