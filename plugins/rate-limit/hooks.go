package ratelimit

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/Authula/authula/models"
)

func (p *RateLimitPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookOnRequest,
			Handler: p.checkEndpointRateLimitHook(),
			Order:   0,
		},
		{
			Stage:   models.HookBefore,
			Handler: p.checkRateLimitRuleHook(),
			Order:   15, // This must run after any plugin that sets the ContextRateLimitRule value, so it should have a higher order than those plugins
		},
		{
			Stage:   models.HookAfter,
			Handler: p.handleStoreRateLimitRuleHook(),
			Order:   0,
		},
	}
}

func (p *RateLimitPlugin) checkEndpointRateLimitHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		validHttpMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
		if !slices.Contains(validHttpMethods, reqCtx.Request.Method) {
			return nil
		}

		key := p.pluginConfig.Prefix + reqCtx.ClientIP
		window := p.pluginConfig.Window
		max := p.pluginConfig.Max

		if rule, exists := p.pluginConfig.CustomRules[reqCtx.Path]; exists {
			if len(rule.Methods) > 0 && !slices.Contains(rule.Methods, reqCtx.Request.Method) {
				// rule doesn't apply to this method; use defaults
			} else if rule.Disabled {
				return nil
			} else {
				if rule.Window > 0 {
					window = rule.Window
				}
				if rule.Max > 0 {
					max = rule.Max
				}
			}
		}

		allowed, count, resetTime, err := p.provider.CheckAndIncrement(ctx, key, window, max)
		if err != nil {
			p.logger.Error("failed to check rate limit", "error", err, "key", key)
			// On error, allow the request to proceed (fail-open)
			return nil
		}

		return p.executeRateLimit(reqCtx, max, count, resetTime, allowed)
	}
}

func (p *RateLimitPlugin) checkRateLimitRuleHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		rawValue, ok := reqCtx.Values[models.ContextRateLimitRule.String()]
		if !ok || rawValue == nil {
			return nil
		}

		ruleCtx, ok := getRateLimitRuleContext(rawValue)
		if !ok || ruleCtx.Key == "" || ruleCtx.WindowSeconds <= 0 || ruleCtx.MaxRequests <= 0 {
			return nil
		}

		window := time.Duration(ruleCtx.WindowSeconds) * time.Second
		allowed, count, resetTime, err := p.provider.CheckAndIncrement(ctx, ruleCtx.Key, window, ruleCtx.MaxRequests)
		if err != nil {
			p.logger.Error("failed to check stored rate limit rule", "key", ruleCtx.Key, "error", err)
			return nil
		}

		return p.executeRateLimit(reqCtx, ruleCtx.MaxRequests, count, resetTime, allowed)
	}
}

func (p *RateLimitPlugin) executeRateLimit(reqCtx *models.RequestContext, maxRequests int, count int, resetTime time.Time, allowed bool) error {
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
	remaining := max(maxRequests-count, 0)
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

	if !allowed {
		retryAfter := int(time.Until(resetTime).Seconds())
		reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
		payload := map[string]any{
			"message":     "rate limit exceeded",
			"retry_after": retryAfter,
			"limit":       maxRequests,
			"remaining":   0,
		}
		reqCtx.SetJSONResponse(http.StatusTooManyRequests, payload)
		reqCtx.Handled = true
	}

	return nil
}

func (p *RateLimitPlugin) handleStoreRateLimitRuleHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		rawValue, ok := reqCtx.Values[models.ContextRateLimitRule.String()]
		if !ok || rawValue == nil {
			return nil
		}

		ruleCtx, ok := getRateLimitRuleContext(rawValue)
		if !ok || ruleCtx.Key == "" || ruleCtx.WindowSeconds <= 0 || ruleCtx.MaxRequests <= 0 {
			return nil
		}

		_, _, _, err := p.provider.GetRule(ctx, ruleCtx.Key)
		if err != nil {
			p.logger.Error("failed to get existing rate limit rule", "key", ruleCtx.Key, "error", err)
			return nil
		}
		window := time.Duration(ruleCtx.WindowSeconds) * time.Second
		err = p.provider.SetRule(ctx, ruleCtx.Key, window, ruleCtx.MaxRequests)
		if err != nil {
			p.logger.Error("failed to store rate limit rule", "key", ruleCtx.Key, "error", err)
			return nil
		}

		return nil
	}
}

func getRateLimitRuleContext(value any) (models.RateLimitRuleContext, bool) {
	switch typed := value.(type) {
	case models.RateLimitRuleContext:
		return typed, true
	case *models.RateLimitRuleContext:
		if typed == nil {
			return models.RateLimitRuleContext{}, false
		}
		return *typed, true
	default:
		return models.RateLimitRuleContext{}, false
	}
}
