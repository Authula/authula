package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type CorePluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`

	// DatabaseHooks allows you to hook into database operations for users, accounts, sessions, and verifications.
	DatabaseHooks *CoreDatabaseHooks `json:"-" toml:"-"`
}

type CoreDatabaseHooks struct {
	Users         *UserDatabaseHooksConfig
	Accounts      *AccountDatabaseHooksConfig
	Sessions      *SessionDatabaseHooksConfig
	Verifications *VerificationDatabaseHooksConfig
}

type UserDatabaseHooksConfig struct {
	BeforeCreate func(user *models.User) error
	AfterCreate  func(user models.User) error
	BeforeUpdate func(user *models.User) error
	AfterUpdate  func(user models.User) error
}

type AccountDatabaseHooksConfig struct {
	BeforeCreate func(account *models.Account) error
	AfterCreate  func(account models.Account) error
	BeforeUpdate func(account *models.Account) error
	AfterUpdate  func(account models.Account) error
}

type SessionDatabaseHooksConfig struct {
	BeforeCreate func(session *models.Session) error
	AfterCreate  func(session models.Session) error
}

type VerificationDatabaseHooksConfig struct {
	BeforeCreate func(verification *models.Verification) error
	AfterCreate  func(verification models.Verification) error
}

type DatabaseHookRunner struct {
	Hooks *CoreDatabaseHooks
}

type HealthCheckResult struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type GetMeResult struct {
	User *models.User
}

type GetMeResponse struct {
	User *models.User `json:"user"`
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
