package handlers

import (
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type VerifyApiKeyHandler struct {
	Service services.ApiKeyService
}

func (h *VerifyApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var req types.VerifyApiKeyRequest
		if err := util.ParseJSON(r, &req); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}
		if err := req.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		result, err := h.Service.Verify(ctx, req)
		if err != nil {
			internalerrors.HandleError(err, reqCtx)
			return
		}

		if !result.Valid {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "invalid or expired API key"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.VerifyApiKeyResponse{ApiKey: result.ApiKey})
	}
}
