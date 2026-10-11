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
	ErrNoSessionToSignOut = errors.New("no session to sign out: provide session_id or sign_out_all")
)

func StatusFor(err error) (int, bool) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, true
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
	status := http.StatusBadRequest
	if mapped, ok := StatusFor(err); ok {
		status = mapped
	}
	reqCtx.SetJSONResponse(status, map[string]any{"message": err.Error()})
	reqCtx.Handled = true
}
