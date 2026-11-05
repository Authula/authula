package services

import "errors"

var (
	ErrAccountNotFound            = errors.New("account not found")
	ErrAccountLinkedToAnotherUser = errors.New("account linked to another user")
	ErrTokenExpired               = errors.New("oauth token expired")
	ErrEmailInUse                 = errors.New("email already in use")
	ErrEmailRequired              = errors.New("email is required")
)
