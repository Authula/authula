package handlers

import (
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
	"github.com/Authula/authula/plugins/api-key/usecases"
	"github.com/Authula/authula/util"
)

type VerifyApiKeyHandler struct {
	UseCases *usecases.UseCases
}

func (h *VerifyApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var req types.VerifyApiKeyRequest
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

		result, err := h.UseCases.Verify(ctx, req)
		if err != nil {
			coreerrors.HandleError(err, reqCtx)
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
