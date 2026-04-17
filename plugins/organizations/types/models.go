package types

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type Organization struct {
	bun.BaseModel `bun:"table:organizations"`

	ID        string          `json:"id" bun:"column:id,pk"`
	OwnerID   string          `json:"owner_id" bun:"column:owner_id"`
	Name      string          `json:"name" bun:"column:name"`
	Slug      string          `json:"slug" bun:"column:slug"`
	Logo      *string         `json:"logo" bun:"column:logo"`
	Metadata  json.RawMessage `json:"metadata" bun:"column:metadata"`
	CreatedAt time.Time       `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt time.Time       `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationInvitationStatus string

const (
	OrganizationInvitationStatusPending  OrganizationInvitationStatus = "pending"
	OrganizationInvitationStatusAccepted OrganizationInvitationStatus = "accepted"
	OrganizationInvitationStatusRejected OrganizationInvitationStatus = "rejected"
	OrganizationInvitationStatusRevoked  OrganizationInvitationStatus = "revoked"
	OrganizationInvitationStatusExpired  OrganizationInvitationStatus = "expired"
)

type OrganizationInvitation struct {
	bun.BaseModel `bun:"table:organization_invitations"`

	ID             string                       `json:"id" bun:"column:id,pk"`
	Email          string                       `json:"email" bun:"column:email"`
	InviterID      string                       `json:"inviter_id" bun:"column:inviter_id"`
	OrganizationID string                       `json:"organization_id" bun:"column:organization_id"`
	Role           string                       `json:"role" bun:"column:role"`
	Status         OrganizationInvitationStatus `json:"status" bun:"column:status"`
	ExpiresAt      time.Time                    `json:"expires_at" bun:"column:expires_at"`
	CreatedAt      time.Time                    `json:"created_at" bun:"column:created_at,default:current_timestamp"`
}

type OrganizationMember struct {
	bun.BaseModel `bun:"table:organization_members"`

	ID             string    `json:"id" bun:"column:id,pk"`
	OrganizationID string    `json:"organization_id" bun:"column:organization_id"`
	UserID         string    `json:"user_id" bun:"column:user_id"`
	Role           string    `json:"role" bun:"column:role"`
	CreatedAt      time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationTeam struct {
	bun.BaseModel `bun:"table:organization_teams"`

	ID             string          `json:"id" bun:"column:id,pk"`
	OrganizationID string          `json:"organization_id" bun:"column:organization_id"`
	Name           string          `json:"name" bun:"column:name"`
	Slug           string          `json:"slug" bun:"column:slug"`
	Description    *string         `json:"description" bun:"column:description"`
	Metadata       json.RawMessage `json:"metadata" bun:"column:metadata"`
	CreatedAt      time.Time       `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time       `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationTeamMember struct {
	bun.BaseModel `bun:"table:organization_team_members"`

	ID        string    `json:"id" bun:"column:id,pk"`
	TeamID    string    `json:"team_id" bun:"column:team_id"`
	MemberID  string    `json:"member_id" bun:"column:member_id"`
	CreatedAt time.Time `json:"created_at" bun:"column:created_at,default:current_timestamp"`
}
