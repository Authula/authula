package handlers

import (
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type UpdateApiKeyHandler struct {
	Service services.ApiKeyService
}

func (h *UpdateApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		id := r.PathValue("id")
		if id == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "id is required"})
			reqCtx.Handled = true
			return
		}

		var req types.UpdateApiKeyRequest
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

		apiKey, err := h.Service.Update(ctx, id, req)
		if err != nil {
			internalerrors.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.UpdateApiKeyResponse{ApiKey: apiKey})
	}
}
