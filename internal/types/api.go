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
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type SignOutRequest struct {
	SessionID  *string `json:"session_id,omitempty"`
	SignOutAll *bool   `json:"sign_out_all,omitempty"`
}

func (req *SignOutRequest) Validate() error {
	if req.SessionID != nil && strings.TrimSpace(*req.SessionID) == "" {
		return errors.New("session_id cannot be empty if provided")
	}
	return nil
}

type SignOutResponse struct {
	Message string `json:"message"`
}

type SignOutResult struct {
	Message string `json:"message"`
}
