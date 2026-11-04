package services

import "errors"

var (
	// ErrAccountNotFound is returned when an account ID does not resolve to a stored account.
	ErrAccountNotFound = errors.New("account not found")
	// ErrAccountLinkedToAnotherUser is returned when an OAuth2 identity already belongs
	// to a different local user.
	ErrAccountLinkedToAnotherUser = errors.New("account linked to another user")
	// ErrTokenExpired is returned when a caller tries to persist an already expired token.
	ErrTokenExpired = errors.New("oauth token expired")
	// ErrEmailInUse is returned when creating a user whose email already exists.
	ErrEmailInUse = errors.New("email already in use")
	// ErrEmailRequired is returned when creating a user without an email address.
	ErrEmailRequired = errors.New("email is required")
)
