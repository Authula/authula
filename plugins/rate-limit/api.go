package ratelimit

import (
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

func SetRuleContext(reqCtx *models.RequestContext, ruleCtx models.RateLimitRuleContext) {
	reqCtx.Values[models.ContextRateLimitRule.String()] = ruleCtx
}

func SetRuleContextFromRecord(reqCtx *models.RequestContext, record *types.RateLimitRuleRecord) {
	if record == nil {
		return
	}

	SetRuleContext(reqCtx, models.RateLimitRuleContext{
		Key:           record.Key,
		WindowSeconds: record.WindowSeconds,
		MaxRequests:   record.MaxRequests,
	})
}
