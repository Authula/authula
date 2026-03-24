package events

import "time"

type OrganizationInvitationCreatedEvent struct {
	ID               string    `json:"id"`
	InvitationID     string    `json:"invitation_id"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	InviteeEmail     string    `json:"invitee_email"`
	InviterID        string    `json:"inviter_id"`
	Role             string    `json:"role"`
	ExpiresAt        time.Time `json:"expires_at"`
	RedirectURL      string    `json:"redirect_url,omitempty"`
}
