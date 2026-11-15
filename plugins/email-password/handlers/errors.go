package handlers

import (
	"errors"
	"net/http"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
)

// writeUseCaseError maps known client errors to their HTTP status and keeps
// unexpected failures as 500.
func writeUseCaseError(reqCtx *models.RequestContext, err error) {
	if errors.Is(err, constants.ErrInvalidOrExpiredToken) {
		reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return
	}
	coreerrors.HandleErrorWithFallback(err, reqCtx, http.StatusInternalServerError)
}
