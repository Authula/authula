package constants

import "errors"

var (
	ErrBadRequest             = errors.New("bad request")
	ErrUnprocessableEntity    = errors.New("unprocessable entity")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrNotFound               = errors.New("not found")
	ErrConflict               = errors.New("conflict")
	ErrCannotUpdateSystemRole = errors.New("cannot update system role")
)
