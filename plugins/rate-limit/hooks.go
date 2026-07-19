package ratelimit

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/Authula/authula/util"
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
			Order:   15,
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

		window := p.pluginConfig.Window
		max := p.pluginConfig.Max

		var identifier string
		if p.pluginConfig.HashClientIP {
			identifier = util.SHA256Hex(reqCtx.ClientIP)
		} else {
			identifier = normalizeIP(reqCtx.ClientIP)
		}

		key := p.pluginConfig.Prefix + hashTagKey(identifier)

		patterns := []string{""}
		if reqCtx.Route != nil && reqCtx.Route.Pattern != "" {
			patterns = append(patterns, reqCtx.Route.Pattern)
		}
		patterns = append(patterns, reqCtx.Request.Method+":"+reqCtx.Path)

		matched := false

		for _, pattern := range patterns {
			if pattern == "" {
				continue
			}

			var rule *types.RateLimitRule
			hashInput := ""

			if rateLimitRule, exists := p.pluginConfig.CustomRules[pattern]; exists {
				rule = &rateLimitRule
				hashInput = pattern
			}

			if rule == nil {
				normPattern := normalizeParams(pattern)
				for rateLimitRuleKey, rateLimitRule := range p.pluginConfig.CustomRules {
					if normalizeParams(rateLimitRuleKey) == normPattern && segmentsMatch(reqCtx.Path, rateLimitRuleKey) {
						rule = &rateLimitRule
						hashInput = pattern
						break
					}
				}
			}

			if rule == nil {
				if pathOnly, ok := strings.CutPrefix(pattern, reqCtx.Request.Method+":"); ok {
					if r, exists := p.pluginConfig.CustomRules[pathOnly]; exists {
						rule = &r
						hashInput = pathOnly
					}
				}
			}

			if rule == nil {
				if pathOnly, ok := strings.CutPrefix(pattern, reqCtx.Request.Method+":"); ok {
					normPathOnly := normalizeParams(pathOnly)
					for rateLimitRuleKey, rateLimitRule := range p.pluginConfig.CustomRules {
						if normalizeParams(rateLimitRuleKey) == normPathOnly && segmentsMatch(reqCtx.Path, rateLimitRuleKey) {
							rule = &rateLimitRule
							hashInput = pathOnly
							break
						}
					}
				}
			}

			if rule != nil {
				if rule.Disabled {
					return nil
				}
				if rule.Window > 0 {
					window = rule.Window
				}
				if rule.Max > 0 {
					max = rule.Max
				}
				key = key + ":" + util.SHA256Hex(hashInput)
				matched = true
				break
			}
		}

		if !matched && reqCtx.Route != nil && reqCtx.Route.Pattern != "" && strings.ContainsAny(reqCtx.Route.Pattern, "{*") {
			window = p.pluginConfig.Window
			max = p.pluginConfig.Max
			key = key + ":" + util.SHA256Hex(reqCtx.Route.Pattern)
		}

		allowed, count, resetTime, err := p.provider.CheckAndIncrement(ctx, key, window, max)
		if err != nil {
			p.logger.Error("failed to check rate limit", "error", err, "key", key)
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

func segmentsMatch(requestPath, ruleKey string) bool {
	rulePath := ruleKey
	if _, after, ok := strings.Cut(ruleKey, ":"); ok {
		rulePath = after
	}
	reqSegs := strings.FieldsFunc(requestPath, func(r rune) bool { return r == '/' })
	ruleSegs := strings.FieldsFunc(rulePath, func(r rune) bool { return r == '/' })
	return len(reqSegs) == len(ruleSegs)
}

func normalizeParams(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			if j := strings.IndexByte(s[i:], '}'); j >= 0 {
				b.WriteString("{*}")
				i += j
				continue
			}
		}
		if s[i] == '*' {
			b.WriteString("{*}")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func hashTagKey(identifier string) string {
	return "{" + identifier + "}"
}

func normalizeIP(ip string) string {
	if strings.Contains(ip, ":") {
		return "[" + ip + "]"
	}
	return ip
}
