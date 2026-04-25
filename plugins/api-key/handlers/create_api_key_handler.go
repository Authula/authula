package handlers

import (
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type CreateApiKeyHandler struct {
	Service services.ApiKeyService
}

func (h *CreateApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var req types.CreateApiKeyRequest
		if err := util.ParseJSON(r, &req); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := req.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		result, err := h.Service.Create(ctx, req)
		if err != nil {
			internalerrors.HandleError(err, reqCtx)
			return
		}

		if req.RateLimitEnabled != nil && *req.RateLimitEnabled {
			reqCtx.Values[models.ContextRateLimitRule.String()] = &models.RateLimitRuleContext{
				Key:           result.ApiKey.KeyHash,
				WindowSeconds: *req.RateLimitTimeWindow,
				MaxRequests:   *req.RateLimitMaxRequests,
			}
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.CreateApiKeyResponse{
			RawApiKey: result.RawApiKey,
			ApiKey:    result.ApiKey,
		})
	}
}
