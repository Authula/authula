package handlers

import (
	"net/http"
	"strings"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	"github.com/Authula/authula/util"
)

type AuthorizeHandler struct {
	UseCase usecases.AuthorizeUseCase
	Cookie  CookieOptions
	Logger  models.Logger
}

func (h *AuthorizeHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		providerID := strings.TrimSpace(r.PathValue("provider"))
		if providerID == "" {
			WriteError(reqCtx, h.Logger, constants.ErrProviderNotFound)
			reqCtx.Handled = true
			return
		}

		req := &types.AuthorizeRequest{
			ProviderID: providerID,
			RedirectTo: util.GetQueryString(r, "redirect_to", ""),
		}
		if reqCtx.Actor != nil && reqCtx.Actor.Type == models.ActorUser {
			req.ActorID = reqCtx.Actor.ID
		}

		resp, err := h.UseCase.Authorize(ctx, req)
		if err != nil {
			WriteError(reqCtx, h.Logger, err)
			reqCtx.Handled = true
			return
		}

		h.Cookie.setBinding(reqCtx.ResponseWriter, resp.Binding)

		reqCtx.SetJSONResponse(http.StatusOK, &types.AuthorizeResponse{
			AuthURL: resp.AuthorizationURL,
		})
		if reqCtx.ResponseHeaders == nil {
			reqCtx.ResponseHeaders = make(http.Header)
		}
		reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	}
}
