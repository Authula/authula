package types

import (
	"errors"
	"strings"

	"github.com/Authula/authula/models"
)

type GetMeResult struct {
	User    *models.User
	Session *models.Session
}

type GetMeResponse struct {
	User    *models.User    `json:"user" required:"true" nullable:"false"`
	Session *models.Session `json:"session" required:"true" nullable:"false"`
}

type SignOutRequest struct {
	SessionID  *string `json:"session_id,omitempty" nullable:"true"`
	SignOutAll *bool   `json:"sign_out_all,omitempty" nullable:"true"`
}

func (req *SignOutRequest) Validate() error {
	if req.SessionID != nil && strings.TrimSpace(*req.SessionID) == "" {
		return errors.New("session_id cannot be empty if provided")
	}
	return nil
}

type SignOutResult struct {
	Message string
}

type SignOutResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}
