package handlers

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
)

// WriteError maps OAuth2 domain errors to safe HTTP responses. Internal details are
// logged but never returned to the client.
func WriteError(reqCtx *models.RequestContext, logger models.Logger, err error) {
	status, message := statusFor(err)

	if logger != nil {
		logger.Error("oauth2 request failed", "status", status, "error", err)
	}

	reqCtx.SetJSONResponse(status, map[string]string{"message": message})
	if reqCtx.ResponseHeaders == nil {
		reqCtx.ResponseHeaders = make(http.Header)
	}
	reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
}

func statusFor(err error) (int, string) {
	switch {
	case errors.Is(err, constants.ErrProviderNotFound):
		return http.StatusNotFound, "provider not found"
	case errors.Is(err, constants.ErrAccessDenied):
		return http.StatusUnauthorized, "authorization denied"
	case errors.Is(err, constants.ErrUnauthorized), errors.Is(err, constants.ErrSessionNotFound):
		return http.StatusUnauthorized, "authentication required"
	case errors.Is(err, constants.ErrAccountConflict),
		errors.Is(err, constants.ErrAccountLinkedToAnotherUser),
		errors.Is(err, constants.ErrEmailInUse):
		return http.StatusConflict, "account conflict"
	case errors.Is(err, constants.ErrEmailRequired),
		errors.Is(err, constants.ErrEmailNotVerified),
		errors.Is(err, constants.ErrProviderAccountIDRequired),
		errors.Is(err, constants.ErrInvalidState),
		errors.Is(err, constants.ErrStateProviderMismatch),
		errors.Is(err, constants.ErrInvalidCode),
		errors.Is(err, constants.ErrPKCEVerifierMissing),
		errors.Is(err, constants.ErrInvalidRedirectTo):
		return http.StatusUnprocessableEntity, "invalid request"
	case errors.Is(err, constants.ErrProviderFailure), errors.Is(err, constants.ErrTokenExpired):
		return http.StatusBadGateway, "provider request failed"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
