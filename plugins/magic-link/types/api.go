package types

import (
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
)

type SignInRequest struct {
	Email       string  `json:"email"`
	Name        *string `json:"name,omitempty"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (r *SignInRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return internalerrors.ErrEmailRequired
	}
	return nil
}

type SignInResult struct {
	Token string
}

type SignInResponse struct {
	Message string `json:"message"`
}

type VerifyResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

type ExchangeRequest struct {
	Token string `json:"token"`
}

func (r *ExchangeRequest) Validate() error {
	if strings.TrimSpace(r.Token) == "" {
		return internalerrors.ErrTokenRequired
	}
	return nil
}

type ExchangeResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

type ExchangeResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}
