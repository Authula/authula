package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

// AuthorizeUseCase is the handler-facing contract for starting an OAuth2 flow.
type AuthorizeUseCase interface {
	Authorize(ctx context.Context, req *types.AuthorizeRequest) (*usecases.AuthorizeResult, error)
}

type AuthorizeHandler struct {
	UseCase AuthorizeUseCase
	Cookie  CookieOptions
	Logger  models.Logger
}

func (h *AuthorizeHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, ok := models.GetRequestContext(ctx)
		if !ok || reqCtx == nil {
			http.Error(w, "request context missing", http.StatusInternalServerError)
			return
		}

		providerID := strings.TrimSpace(r.PathValue("provider"))
		if providerID == "" {
			WriteError(reqCtx, h.Logger, constants.ErrProviderNotFound)
			reqCtx.Handled = true
			return
		}

		req := &types.AuthorizeRequest{
			ProviderID: providerID,
			RedirectTo: r.URL.Query().Get("redirect_to"),
		}
		if reqCtx.Actor != nil && reqCtx.Actor.Type == models.ActorUser {
			req.ActorID = reqCtx.Actor.ID
		}

		resp, err := h.UseCase.Authorize(ctx, req)
		if err != nil {
			h.Cookie.clearFlowCookies(reqCtx.ResponseWriter)
			WriteError(reqCtx, h.Logger, err)
			reqCtx.Handled = true
			return
		}

		h.Cookie.set(reqCtx.ResponseWriter, constants.CookieState, resp.StateCookie)
		h.Cookie.set(reqCtx.ResponseWriter, constants.CookieRedirectTo, resp.RedirectCookie)
		if resp.VerifierCookie != nil {
			h.Cookie.set(reqCtx.ResponseWriter, constants.CookieVerifier, *resp.VerifierCookie)
		} else {
			h.Cookie.clear(reqCtx.ResponseWriter, constants.CookieVerifier)
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.AuthorizeResponse{
			AuthURL: resp.AuthorizationURL,
		})
		if reqCtx.ResponseHeaders == nil {
			reqCtx.ResponseHeaders = make(http.Header)
		}
		reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	}
}
