package handlers

import (
	"errors"
	"net/http"
	"testing"

	coreerrors "github.com/Authula/authula/core/errors"
)

func TestMapAdminHttpErrorMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: "internal server error"},
		{name: "bad request", err: coreerrors.ErrBadRequest, want: "bad request"},
		{name: "unauthorized", err: coreerrors.ErrUnauthorized, want: "unauthorized"},
		{name: "forbidden", err: coreerrors.ErrForbidden, want: "forbidden"},
		{name: "insufficient permissions", err: coreerrors.ErrInsufficientPermissions, want: "forbidden"},
		{name: "not found", err: coreerrors.ErrNotFound, want: "not found"},
		{name: "conflict", err: coreerrors.ErrConflict, want: "conflict"},
		{name: "unprocessable entity", err: coreerrors.ErrUnprocessableEntity, want: "unprocessable entity"},
		{name: "wrapped known error", err: errors.Join(errors.New("context"), coreerrors.ErrConflict), want: "conflict"},
		{name: "unknown error uses message", err: errors.New("boom"), want: "boom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapAdminHttpErrorMessage(tt.err); got != tt.want {
				t.Fatalf("mapAdminHttpErrorMessage(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestMapAdminHttpErrorStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "known error", err: coreerrors.ErrNotFound, want: http.StatusNotFound},
		{name: "unknown error", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapAdminHttpErrorStatus(tt.err); got != tt.want {
				t.Fatalf("mapAdminHttpErrorStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestRespondErrorHelpers_NilContextIsNoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		respond func(err error)
	}{
		{name: "state", respond: func(err error) { respondStateError(nil, err) }},
		{name: "accounts", respond: func(err error) { respondAccountsError(nil, err) }},
		{name: "users", respond: func(err error) { respondUsersError(nil, err) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.respond(coreerrors.ErrBadRequest)
		})
	}
}
