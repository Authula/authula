package types

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
)

type Impersonation struct {
	bun.BaseModel `bun:"table:admin_impersonations"`

	ID                     string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	ActorUserID            string     `json:"actor_user_id" required:"true" nullable:"false" bun:"column:actor_user_id"`
	TargetUserID           string     `json:"target_user_id" required:"true" nullable:"false" bun:"column:target_user_id"`
	ActorSessionID         *string    `json:"actor_session_id" nullable:"true" bun:"column:actor_session_id"`
	ImpersonationSessionID *string    `json:"impersonation_session_id" nullable:"true" bun:"column:impersonation_session_id"`
	Reason                 string     `json:"reason" required:"true" nullable:"false" bun:"column:reason"`
	StartedAt              time.Time  `json:"started_at" required:"true" nullable:"false" bun:"column:started_at"`
	ExpiresAt              time.Time  `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	EndedAt                *time.Time `json:"ended_at" nullable:"true" bun:"column:ended_at"`
	EndedByUserID          *string    `json:"ended_by_user_id" nullable:"true" bun:"column:ended_by_user_id"`
	CreatedAt              time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserState struct {
	bun.BaseModel `bun:"table:admin_user_states"`

	UserID         string     `json:"user_id" required:"true" nullable:"false" bun:"column:user_id,pk"`
	Banned         bool       `json:"banned" required:"true" nullable:"false" bun:"column:banned"`
	BannedAt       *time.Time `json:"banned_at" nullable:"true" bun:"column:banned_at"`
	BannedUntil    *time.Time `json:"banned_until" nullable:"true" bun:"column:banned_until"`
	BannedReason   *string    `json:"banned_reason" nullable:"true" bun:"column:banned_reason"`
	BannedByUserID *string    `json:"banned_by_user_id" nullable:"true" bun:"column:banned_by_user_id"`
	CreatedAt      time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type AdminSessionState struct {
	bun.BaseModel `bun:"table:admin_session_states"`

	SessionID              string     `json:"session_id" required:"true" nullable:"false" bun:"column:session_id,pk"`
	RevokedAt              *time.Time `json:"revoked_at" nullable:"true" bun:"column:revoked_at"`
	RevokedReason          *string    `json:"revoked_reason" nullable:"true" bun:"column:revoked_reason"`
	RevokedByUserID        *string    `json:"revoked_by_user_id" nullable:"true" bun:"column:revoked_by_user_id"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id" nullable:"true" bun:"column:impersonator_user_id"`
	ImpersonatorSessionID  *string    `json:"impersonator_session_id" nullable:"true" bun:"column:impersonator_session_id"`
	ImpersonationReason    *string    `json:"impersonation_reason" nullable:"true" bun:"column:impersonation_reason"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at" nullable:"true" bun:"column:impersonation_expires_at"`
	CreatedAt              time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserSession struct {
	Session models.Session     `json:"session" required:"true" nullable:"false"`
	State   *AdminSessionState `json:"state,omitempty" nullable:"true"`
}
