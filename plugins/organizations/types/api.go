package types

import (
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
)

type OrganizationID struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
}

type InvitationID struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	InvitationID   string `path:"invitation_id" json:"invitation_id" required:"true" nullable:"false"`
}

type MemberID struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	MemberID       string `path:"member_id" json:"member_id" required:"true" nullable:"false"`
}

type TeamID struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	TeamID         string `path:"team_id" json:"team_id" required:"true" nullable:"false"`
}

type TeamMemberID struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	TeamID         string `path:"team_id" json:"team_id" required:"true" nullable:"false"`
	MemberID       string `path:"member_id" json:"member_id" required:"true" nullable:"false"`
}

type ListOrganizationMembersRequest struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	Page           int    `query:"page" json:"page,omitempty" nullable:"false"`
	Limit          int    `query:"limit" json:"limit,omitempty" nullable:"false"`
}

type ListOrganizationTeamMembersRequest struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	TeamID         string `path:"team_id" json:"team_id" required:"true" nullable:"false"`
	Page           int    `query:"page" json:"page,omitempty" nullable:"false"`
	Limit          int    `query:"limit" json:"limit,omitempty" nullable:"false"`
}

type AcceptOrganizationInvitationQuery struct {
	OrganizationID string `path:"organization_id" json:"organization_id" required:"true" nullable:"false"`
	InvitationID   string `path:"invitation_id" json:"invitation_id" required:"true" nullable:"false"`
	RedirectURL    string `query:"redirect_url" json:"redirect_url,omitempty" nullable:"true"`
}

type CreateOrganizationRequest struct {
	Name     string         `json:"name" required:"true" nullable:"false"`
	Role     string         `json:"role" required:"true" nullable:"false"`
	Slug     *string        `json:"slug,omitempty" nullable:"true"`
	Logo     *string        `json:"logo,omitempty" nullable:"true"`
	Metadata map[string]any `json:"metadata,omitempty" nullable:"true"`
}

func (r *CreateOrganizationRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if strings.TrimSpace(r.Role) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Logo != nil {
		value := strings.TrimSpace(*r.Logo)
		r.Logo = &value
	}
	return nil
}

type UpdateOrganizationRequest struct {
	Name     *string        `json:"name,omitempty" nullable:"true"`
	Slug     *string        `json:"slug,omitempty" nullable:"true"`
	Logo     *string        `json:"logo,omitempty" nullable:"true"`
	Metadata map[string]any `json:"metadata,omitempty" nullable:"true"`
}

func (r *UpdateOrganizationRequest) Validate() error {
	if r.Name != nil {
		value := strings.TrimSpace(*r.Name)
		if value == "" {
			return internalerrors.ErrUnprocessableEntity
		}
		r.Name = &value
	}
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Logo != nil {
		value := strings.TrimSpace(*r.Logo)
		r.Logo = &value
	}
	return nil
}

type CreateOrganizationInvitationRequest struct {
	Email       string `json:"email" required:"true" nullable:"false"`
	Role        string `json:"role" required:"true" nullable:"false"`
	RedirectURL string `json:"redirect_url,omitempty" nullable:"true"`
}

func (r *CreateOrganizationInvitationRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if strings.TrimSpace(r.Role) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type AddOrganizationMemberRequest struct {
	UserID string `json:"user_id" required:"true" nullable:"false"`
	Role   string `json:"role" required:"true" nullable:"false"`
}

func (r *AddOrganizationMemberRequest) Validate() error {
	if strings.TrimSpace(r.UserID) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if strings.TrimSpace(r.Role) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type UpdateOrganizationMemberRequest struct {
	Role string `json:"role" required:"true" nullable:"false"`
}

func (r *UpdateOrganizationMemberRequest) Validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type CreateOrganizationTeamRequest struct {
	Name        string         `json:"name" required:"true" nullable:"false"`
	Slug        *string        `json:"slug,omitempty" nullable:"true"`
	Description *string        `json:"description,omitempty" nullable:"true"`
	Metadata    map[string]any `json:"metadata,omitempty" nullable:"true"`
}

func (r *CreateOrganizationTeamRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Description != nil {
		value := strings.TrimSpace(*r.Description)
		r.Description = &value
	}
	return nil
}

type UpdateOrganizationTeamRequest struct {
	Name        string         `json:"name" required:"true" nullable:"false"`
	Slug        *string        `json:"slug,omitempty" nullable:"true"`
	Description *string        `json:"description,omitempty" nullable:"true"`
	Metadata    map[string]any `json:"metadata,omitempty" nullable:"true"`
}

func (r *UpdateOrganizationTeamRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Description != nil {
		value := strings.TrimSpace(*r.Description)
		r.Description = &value
	}
	return nil
}

type AddOrganizationTeamMemberRequest struct {
	MemberID string `json:"member_id" required:"true" nullable:"false"`
}

func (r *AddOrganizationTeamMemberRequest) Validate() error {
	if strings.TrimSpace(r.MemberID) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type AcceptOrganizationInvitationRequest struct {
	RedirectURL *string `json:"redirect_url,omitempty" nullable:"true"`
}

func (r *AcceptOrganizationInvitationRequest) Validate() error {
	if r.RedirectURL != nil {
		value := strings.TrimSpace(*r.RedirectURL)
		r.RedirectURL = &value
	}
	return nil
}

type DeleteOrganizationResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type DeleteOrganizationMemberResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type DeleteOrganizationTeamResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type DeleteOrganizationTeamMemberResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}
