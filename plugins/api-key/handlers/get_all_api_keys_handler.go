package handlers

import (
	"net/http"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type GetAllApiKeysHandler struct {
	Service services.ApiKeyService
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

		resp, err := h.Service.GetAll(ctx, reqCtx.Actor, req)
		if err != nil {
			internalerrors.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, resp)
	}
}
