package handlers

import (
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
	"github.com/Authula/authula/plugins/api-key/usecases"
)

type GetApiKeyHandler struct {
	UseCases *usecases.UseCases
}

func (h *GetApiKeyHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		id := r.PathValue("id")
		if id == "" {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "id is required"})
			reqCtx.Handled = true
			return
		}

		apiKey, err := h.UseCases.GetByID(ctx, reqCtx.Actor, id)
		if err != nil {
			coreerrors.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.GetApiKeyResponse{ApiKey: apiKey})
	}
}
