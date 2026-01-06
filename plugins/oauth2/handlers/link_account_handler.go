package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/usecases"
)

type LinkHandler struct {
	UseCase *usecases.LinkAccountUseCase
}

func NewLinkHandler(useCase *usecases.LinkAccountUseCase) *LinkHandler {
	return &LinkHandler{UseCase: useCase}
}

func (h *LinkHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		providerID := r.PathValue("provider")
		if providerID == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "provider is required",
			})
			reqCtx.Handled = true
			return
		}

		providerAccountID := r.URL.Query().Get("provider_account_id")
		if providerAccountID == "" {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]string{
				"message": "provider_account_id is required",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.LinkAccount(ctx, userID, providerID, providerAccountID)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, result)
	}
}
