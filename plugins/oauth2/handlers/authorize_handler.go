package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type AuthorizeHandler struct {
	UseCase         usecases.AuthorizeUseCase
	RateLimiter     rootservices.RateLimiterService
	RateLimitWindow time.Duration
	RateLimitMax    int
	Cookie          CookieOptions
	Logger          models.Logger
}

func (h *AuthorizeHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if !h.allowRequest(reqCtx) {
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

// allowRequest applies the shared rate limiter before any use-case work.
func (h *AuthorizeHandler) allowRequest(reqCtx *models.RequestContext) bool {
	key := "oauth2:authorize:" + reqCtx.ClientIP
	allowed, count, resetAt, err := h.RateLimiter.CheckAndIncrement(reqCtx.Request.Context(), key, h.RateLimitWindow, h.RateLimitMax)
	if err != nil {
		h.Logger.Error("oauth2 authorize rate limit check failed", "error", err)
		WriteError(reqCtx, h.Logger, fmt.Errorf("rate limiter unavailable: %w", err))
		reqCtx.Handled = true
		return false
	}

	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(h.RateLimitMax))
	remaining := max(h.RateLimitMax-count, 0)
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

	if allowed {
		return true
	}

	retryAfter := max(int(time.Until(resetAt).Seconds()), 1)
	reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
	reqCtx.SetJSONResponse(http.StatusTooManyRequests, map[string]any{
		"message":     "rate limit exceeded",
		"retry_after": retryAfter,
		"limit":       h.RateLimitMax,
		"remaining":   0,
	})
	reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	reqCtx.Handled = true
	return false
}
