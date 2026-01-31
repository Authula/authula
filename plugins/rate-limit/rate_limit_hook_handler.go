package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type RateLimitHookHandler struct {
	config       *models.Config
	logger       models.Logger
	pluginConfig RateLimitPluginConfig
	provider     RateLimitProvider
	customRules  map[string]RateLimitRule
}

func NewRateLimitHookHandler(
	config *models.Config,
	logger models.Logger,
	pluginConfig RateLimitPluginConfig,
	provider RateLimitProvider,
) *RateLimitHookHandler {
	return &RateLimitHookHandler{
		config:       config,
		logger:       logger,
		provider:     provider,
		pluginConfig: pluginConfig,
		customRules:  pluginConfig.CustomRules,
	}
}

func (h *RateLimitHookHandler) Handle() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		// Skip rate limiting for OPTIONS preflight requests if CORS is handled externally
		// or by the CORS plugin middleware (which runs later).
		if reqCtx.Request.Method == http.MethodOptions {
			return nil
		}

		key := h.pluginConfig.Prefix + reqCtx.ClientIP

		window := h.pluginConfig.Window
		max := h.pluginConfig.Max

		if rule, exists := h.customRules[reqCtx.Request.RequestURI]; exists {
			if rule.Disabled {
				return nil
			}
			if rule.Window > 0 {
				window = rule.Window
			}
			if rule.Max > 0 {
				max = rule.Max
			}
		}

		allowed, count, resetTime, err := h.provider.CheckAndIncrement(reqCtx.Request.Context(), key, window, max)
		if err != nil {
			h.logger.Error("failed to check rate limit", "error", err, "key", key)
			// On error, allow the request to proceed (fail-open)
			return nil
		}

		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(max))
		remaining := max - count
		if remaining < 0 {
			remaining = 0
		}
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
			payload := map[string]any{
				"message":     "rate limit exceeded",
				"retry_after": retryAfter,
				"limit":       max,
				"remaining":   0,
			}
			reqCtx.SetJSONResponse(http.StatusTooManyRequests, payload)
			reqCtx.Handled = true
		}

		return nil
	}
}
