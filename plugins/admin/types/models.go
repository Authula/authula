package types

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
)

type Impersonation struct {
	bun.BaseModel `bun:"table:admin_impersonations"`

	ID                     string     `json:"id" bun:"column:id,pk"`
	ActorUserID            string     `json:"actor_user_id" bun:"column:actor_user_id"`
	TargetUserID           string     `json:"target_user_id" bun:"column:target_user_id"`
	ActorSessionID         *string    `json:"actor_session_id" bun:"column:actor_session_id"`
	ImpersonationSessionID *string    `json:"impersonation_session_id" bun:"column:impersonation_session_id"`
	Reason                 string     `json:"reason" bun:"column:reason"`
	StartedAt              time.Time  `json:"started_at" bun:"column:started_at"`
	ExpiresAt              time.Time  `json:"expires_at" bun:"column:expires_at"`
	EndedAt                *time.Time `json:"ended_at" bun:"column:ended_at"`
	EndedByUserID          *string    `json:"ended_by_user_id" bun:"column:ended_by_user_id"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserState struct {
	bun.BaseModel `bun:"table:admin_user_states"`

	UserID         string     `json:"user_id" bun:"column:user_id,pk"`
	Banned         bool       `json:"banned" bun:"column:banned"`
	BannedAt       *time.Time `json:"banned_at" bun:"column:banned_at"`
	BannedUntil    *time.Time `json:"banned_until" bun:"column:banned_until"`
	BannedReason   *string    `json:"banned_reason" bun:"column:banned_reason"`
	BannedByUserID *string    `json:"banned_by_user_id" bun:"column:banned_by_user_id"`
	CreatedAt      time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminSessionState struct {
	bun.BaseModel `bun:"table:admin_session_states"`

	SessionID              string     `json:"session_id" bun:"column:session_id,pk"`
	RevokedAt              *time.Time `json:"revoked_at" bun:"column:revoked_at"`
	RevokedReason          *string    `json:"revoked_reason" bun:"column:revoked_reason"`
	RevokedByUserID        *string    `json:"revoked_by_user_id" bun:"column:revoked_by_user_id"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id" bun:"column:impersonator_user_id"`
	ImpersonationReason    *string    `json:"impersonation_reason" bun:"column:impersonation_reason"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at" bun:"column:impersonation_expires_at"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserSession struct {
	Session models.Session     `json:"session"`
	State   *AdminSessionState `json:"state,omitempty"`
}
