package handlers

import (
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type DeleteApiKeyHandler struct {
	Service services.ApiKeyService
}

func (h *DeleteApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		id := r.PathValue("id")
		if id == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "id is required"})
			reqCtx.Handled = true
			return
		}

		if err := h.Service.Delete(ctx, id); err != nil {
			internalerrors.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteApiKeyResponse{Message: "API key deleted successfully"})
	}
}
