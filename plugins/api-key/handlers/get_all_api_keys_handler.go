package handlers

import (
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
	"github.com/Authula/authula/plugins/api-key/usecases"
)

type GetAllApiKeysHandler struct {
	UseCases *usecases.UseCases
}

func (h *GetAllApiKeysHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		page := util.GetQueryInt(r, "page", 1)
		limit := util.GetQueryInt(r, "limit", 0)

		req := types.GetApiKeysRequest{
			Page:  page,
			Limit: limit,
		}

		if ownerType := r.URL.Query().Get("owner_type"); ownerType != "" {
			req.OwnerType = &ownerType
		}
		if ownerID := r.URL.Query().Get("owner_id"); ownerID != "" {
			req.OwnerID = &ownerID
		}

		resp, err := h.UseCases.GetAll(ctx, reqCtx.Actor, req)
		if err != nil {
			coreerrors.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, resp)
	}
}
