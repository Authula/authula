package types

import (
	"time"

	"github.com/swaggest/jsonschema-go"
	"github.com/uptrace/bun"
)

type Organization struct {
	bun.BaseModel `bun:"table:organizations"`

	ID        string         `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	OwnerID   string         `json:"owner_id" required:"true" nullable:"false" bun:"column:owner_id"`
	Name      string         `json:"name" required:"true" nullable:"false" bun:"column:name"`
	Slug      string         `json:"slug" required:"true" nullable:"false" bun:"column:slug"`
	Logo      *string        `json:"logo" nullable:"true" bun:"column:logo"`
	Metadata  map[string]any `json:"metadata" nullable:"true" bun:"column:metadata"`
	CreatedAt time.Time      `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt time.Time      `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationInvitationStatus string

const (
	OrganizationInvitationStatusPending  OrganizationInvitationStatus = "pending"
	OrganizationInvitationStatusAccepted OrganizationInvitationStatus = "accepted"
	OrganizationInvitationStatusRejected OrganizationInvitationStatus = "rejected"
	OrganizationInvitationStatusRevoked  OrganizationInvitationStatus = "revoked"
	OrganizationInvitationStatusExpired  OrganizationInvitationStatus = "expired"
)

func (OrganizationInvitationStatus) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithType(jsonschema.String.Type())
	schema.Enum = []any{
		string(OrganizationInvitationStatusPending),
		string(OrganizationInvitationStatusAccepted),
		string(OrganizationInvitationStatusRejected),
		string(OrganizationInvitationStatusRevoked),
		string(OrganizationInvitationStatusExpired),
	}
	schema.WithDescription("The type of the invitation status")
	return nil
}

type OrganizationInvitation struct {
	bun.BaseModel `bun:"table:organization_invitations"`

	ID             string                       `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	Email          string                       `json:"email" required:"true" nullable:"false" bun:"column:email"`
	InviterID      string                       `json:"inviter_id" required:"true" nullable:"false" bun:"column:inviter_id"`
	OrganizationID string                       `json:"organization_id" required:"true" nullable:"false" bun:"column:organization_id"`
	Role           string                       `json:"role" required:"true" nullable:"false" bun:"column:role"`
	Status         OrganizationInvitationStatus `json:"status" required:"true" nullable:"false" bun:"column:status"`
	ExpiresAt      time.Time                    `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	CreatedAt      time.Time                    `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}

type OrganizationMember struct {
	bun.BaseModel `bun:"table:organization_members"`

	ID             string    `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	OrganizationID string    `json:"organization_id" required:"true" nullable:"false" bun:"column:organization_id"`
	UserID         string    `json:"user_id" required:"true" nullable:"false" bun:"column:user_id"`
	Role           string    `json:"role" required:"true" nullable:"false" bun:"column:role"`
	CreatedAt      time.Time `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationTeam struct {
	bun.BaseModel `bun:"table:organization_teams"`

	ID             string         `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	OrganizationID string         `json:"organization_id" required:"true" nullable:"false" bun:"column:organization_id"`
	Name           string         `json:"name" required:"true" nullable:"false" bun:"column:name"`
	Slug           string         `json:"slug" required:"true" nullable:"false" bun:"column:slug"`
	Description    *string        `json:"description" nullable:"true" bun:"column:description"`
	Metadata       map[string]any `json:"metadata" nullable:"true" bun:"column:metadata"`
	CreatedAt      time.Time      `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time      `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

type OrganizationTeamMember struct {
	bun.BaseModel `bun:"table:organization_team_members"`

	ID        string    `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	TeamID    string    `json:"team_id" required:"true" nullable:"false" bun:"column:team_id"`
	MemberID  string    `json:"member_id" required:"true" nullable:"false" bun:"column:member_id"`
	CreatedAt time.Time `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}
