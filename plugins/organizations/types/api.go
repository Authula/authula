package types

import (
	"encoding/json"
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
)

type CreateOrganizationRequest struct {
	Name     string          `json:"name"`
	Role     string          `json:"role"`
	Slug     *string         `json:"slug,omitempty"`
	Logo     *string         `json:"logo,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
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
	Name     string          `json:"name"`
	Slug     *string         `json:"slug,omitempty"`
	Logo     *string         `json:"logo,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

func (r *UpdateOrganizationRequest) Trim() {
	r.Name = strings.TrimSpace(r.Name)
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Logo != nil {
		value := strings.TrimSpace(*r.Logo)
		r.Logo = &value
	}
}

type CreateOrganizationInvitationRequest struct {
	Email       string `json:"email"`
	Role        string `json:"role"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

func (r *CreateOrganizationInvitationRequest) Trim() {
	r.Email = strings.TrimSpace(r.Email)
	r.Role = strings.TrimSpace(r.Role)
	r.RedirectURL = strings.TrimSpace(r.RedirectURL)
}

type AddOrganizationMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (r *AddOrganizationMemberRequest) Trim() {
	r.UserID = strings.TrimSpace(r.UserID)
	r.Role = strings.TrimSpace(r.Role)
}

type UpdateOrganizationMemberRequest struct {
	Role string `json:"role"`
}

func (r *UpdateOrganizationMemberRequest) Trim() {
	r.Role = strings.TrimSpace(r.Role)
}

type CreateOrganizationTeamRequest struct {
	Name        string          `json:"name"`
	Slug        *string         `json:"slug,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

func (r *CreateOrganizationTeamRequest) Trim() {
	r.Name = strings.TrimSpace(r.Name)
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Description != nil {
		value := strings.TrimSpace(*r.Description)
		r.Description = &value
	}
}

type UpdateOrganizationTeamRequest struct {
	Name        string          `json:"name"`
	Slug        *string         `json:"slug,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

func (r *UpdateOrganizationTeamRequest) Trim() {
	r.Name = strings.TrimSpace(r.Name)
	if r.Slug != nil {
		value := strings.TrimSpace(*r.Slug)
		r.Slug = &value
	}
	if r.Description != nil {
		value := strings.TrimSpace(*r.Description)
		r.Description = &value
	}
}

type AddOrganizationTeamMemberRequest struct {
	MemberID string `json:"member_id"`
}

func (r *AddOrganizationTeamMemberRequest) Trim() {
	r.MemberID = strings.TrimSpace(r.MemberID)
}

type AcceptOrganizationInvitationRequest struct {
	RedirectURL string `json:"redirect_url,omitempty"`
}

func (r *AcceptOrganizationInvitationRequest) Trim() {
	r.RedirectURL = strings.TrimSpace(r.RedirectURL)
}

type DeleteOrganizationResponse struct {
	Message string `json:"message"`
}

type DeleteOrganizationMemberResponse struct {
	Message string `json:"message"`
}

type DeleteOrganizationTeamResponse struct {
	Message string `json:"message"`
}

type DeleteOrganizationTeamMemberResponse struct {
	Message string `json:"message"`
}
