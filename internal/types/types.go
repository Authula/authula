package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type HealthCheckResult struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

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
	SignOutAll bool    `json:"sign_out_all,omitempty"`
}

type SignOutResponse struct {
	Message string `json:"message"`
}

type SignOutResult struct {
	Message string `json:"message"`
}
