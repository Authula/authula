package types

import (
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
)

type MagicLinkSignInRequest struct {
	Email       string  `json:"email" required:"true" nullable:"false"`
	Name        *string `json:"name,omitempty" nullable:"true"`
	CallbackURL *string `json:"callback_url,omitempty" nullable:"true"`
}

type MagicLinkVerifyRequest struct {
	Token       string `query:"token" json:"token" required:"true" nullable:"false"`
	CallbackURL string `query:"callback_url" json:"callback_url,omitempty" nullable:"true"`
}

func (r *MagicLinkVerifyRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	return nil
}

func (r *MagicLinkSignInRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return internalerrors.ErrEmailRequired
	}
	return nil
}

type MagicLinkSignInResult struct {
	Token string
}

type MagicLinkSignInResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type MagicLinkVerifyResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
	Token   string `json:"token,omitempty" nullable:"false"`
}

type MagicLinkExchangeRequest struct {
	Token string `json:"token" required:"true" nullable:"false"`
}

func (r *MagicLinkExchangeRequest) Validate() error {
	if strings.TrimSpace(r.Token) == "" {
		return internalerrors.ErrTokenRequired
	}
	return nil
}

type MagicLinkExchangeResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

type MagicLinkExchangeResponse struct {
	User    *models.User    `json:"user" required:"true" nullable:"false"`
	Session *models.Session `json:"session" required:"true" nullable:"false"`
}
