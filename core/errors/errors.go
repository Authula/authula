package errors

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/models"
)

var (
	// HTTP Errors
	ErrBadRequest              = errors.New("bad request")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrNotFound                = errors.New("not found")
	ErrConflict                = errors.New("conflict")
	ErrUnprocessableEntity     = errors.New("unprocessable entity")
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// User Errors
	ErrEmailRequired = errors.New("email is required")

	// Token Errors
	ErrTokenRequired = errors.New("token is required")

	// Session Errors
	ErrSessionNotFound    = errors.New("session not found")
	ErrNoSessionToSignOut = errors.New("no session to sign out: provide session_id or sign_out_all")

	// Verification Errors
	ErrVerificationNotFound = errors.New("verification not found")
)

func StatusFor(err error) (int, bool) {
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrSessionNotFound):
		return http.StatusUnauthorized, true
	case errors.Is(err, ErrVerificationNotFound):
		return http.StatusBadRequest, true
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrInsufficientPermissions):
		return http.StatusForbidden, true
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, true
	case errors.Is(err, ErrUnprocessableEntity):
		return http.StatusUnprocessableEntity, true
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, true
	default:
		return 0, false
	}
}

func HandleError(err error, reqCtx *models.RequestContext) {
	HandleErrorWithFallback(err, reqCtx, http.StatusBadRequest)
}

// HandleErrorWithFallback writes a consistent error response, using fallbackStatus
// when the error is not one of the recognised sentinels.
func HandleErrorWithFallback(err error, reqCtx *models.RequestContext, fallbackStatus int) {
	status := fallbackStatus
	if mapped, ok := StatusFor(err); ok {
		status = mapped
	}
	reqCtx.SetJSONResponse(status, map[string]any{"message": err.Error()})
	reqCtx.Handled = true
}
