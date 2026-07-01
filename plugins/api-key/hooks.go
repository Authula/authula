package apikey

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
)

func (p *ApiKeyPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Matcher: p.matchApiKeyHeader,
			Handler: p.validateApiKeyHook(),
			Order:   7,
		},
	}
}

func (p *ApiKeyPlugin) matchApiKeyHeader(reqCtx *models.RequestContext) bool {
	return reqCtx.Request.Header.Get(p.config.Header) != ""
}

func (p *ApiKeyPlugin) validateApiKeyHook() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		ctx := reqCtx.Request.Context()

		apiKeyValue := reqCtx.Request.Header.Get(p.config.Header)

		result, err := p.Api.Verify(ctx, types.VerifyApiKeyRequest{Key: apiKeyValue})
		if err != nil {
			p.logger.Error("failed to verify api key", "error", err)
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return nil
		}
		if result == nil || !result.Valid || result.ApiKey == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "invalid api key"})
			reqCtx.Handled = true
			return nil
		}

		apiKey := result.ApiKey
		actor := &models.Actor{}

		switch apiKey.OwnerType {
		case types.OwnerTypeUser:
			user, err := p.userService.GetByID(ctx, apiKey.OwnerID)
			if err != nil {
				p.logger.Error("failed to verify user for api key", "error", err, "user_id", apiKey.OwnerID)
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
				reqCtx.Handled = true
				return nil
			}
			if user == nil {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "api key owner not found"})
				reqCtx.Handled = true
				return nil
			}

			actor.Type = models.ActorUser
			actor.ID = apiKey.OwnerID
			actor.Scopes = apiKey.Permissions

		case types.OwnerTypeOrganization:
			if p.organizationService == nil {
				p.logger.Error("organization service is not available for org api key validation")
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "organization service unavailable"})
				reqCtx.Handled = true
				return nil
			}

			exists, err := p.organizationService.ExistsByID(ctx, apiKey.OwnerID)
			if err != nil {
				p.logger.Error("failed to verify organization for api key", "error", err, "organization_id", apiKey.OwnerID)
				reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
				reqCtx.Handled = true
				return nil
			}
			if !exists {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "api key organization not found"})
				reqCtx.Handled = true
				return nil
			}

			actor.Type = models.ActorMachine
			actor.ID = apiKey.ID
			actor.Scopes = apiKey.Permissions
			actor.Claims = map[string]any{"organization_id": apiKey.OwnerID}

		default:
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "invalid api key owner type"})
			reqCtx.Handled = true
			return nil
		}

		reqCtx.Actor = actor

		apiKey, err = p.Api.RecordLastRequest(ctx, apiKey.ID, time.Now().UTC())
		if err != nil {
			p.logger.Error("failed to update api key", "error", err)
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return nil
		}

		p.applyApiKeyRateLimit(reqCtx, apiKey)

		return nil
	}
}

func (p *ApiKeyPlugin) applyApiKeyRateLimit(reqCtx *models.RequestContext, apiKey *types.ApiKey) {
	ctx := reqCtx.Request.Context()

	if !apiKey.RateLimitEnabled {
		return
	}

	if p.rateLimiterService == nil {
		return
	}

	rule, err := p.rateLimiterService.GetRule(ctx, apiKey.KeyHash)
	if err != nil {
		p.logger.Error(fmt.Sprintf("failed to get rule for API Key: %s", apiKey.ID), "error", err.Error())
		return
	}

	allowed, count, resetAt, err := p.rateLimiterService.CheckAndIncrement(ctx, apiKey.KeyHash, rule.Window, rule.MaxRequests)
	if err != nil {
		p.logger.Error("failed to check api key rate limit", "error", err, "api_key_id", apiKey.ID)
		return
	}

	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(rule.MaxRequests))
	remaining := max(rule.MaxRequests-count, 0)
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	reqCtx.ResponseWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

	if !allowed {
		retryAfter := int(time.Until(resetAt).Seconds())
		reqCtx.ResponseWriter.Header().Set("X-Retry-After", strconv.Itoa(retryAfter))
		reqCtx.SetJSONResponse(http.StatusTooManyRequests, map[string]any{
			"message":     "rate limit exceeded",
			"retry_after": retryAfter,
			"limit":       rule.MaxRequests,
			"remaining":   0,
		})
		reqCtx.Handled = true
	}
}
