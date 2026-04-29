package errors

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/models"
)

var (
	// HTTP Errors
	ErrBadRequest          = errors.New("bad request")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrUnprocessableEntity = errors.New("unprocessable entity")

	// User Errors
	ErrEmailRequired = errors.New("email is required")

	// Token Errors
	ErrTokenRequired = errors.New("token is required")
)

func HandleError(err error, reqCtx *models.RequestContext) {
	status := http.StatusBadRequest
	switch err {
	case ErrUnauthorized:
		status = http.StatusUnauthorized
	case ErrForbidden:
		status = http.StatusForbidden
	case ErrNotFound:
		status = http.StatusNotFound
	case ErrConflict:
		status = http.StatusConflict
	case ErrUnprocessableEntity:
		status = http.StatusUnprocessableEntity
	}
	reqCtx.SetJSONResponse(status, map[string]any{"message": err.Error()})
	reqCtx.Handled = true
}
