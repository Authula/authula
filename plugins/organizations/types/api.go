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

func (r *UpdateOrganizationRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
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

type CreateOrganizationInvitationRequest struct {
	Email       string `json:"email"`
	Role        string `json:"role"`
	RedirectURL string `json:"redirect_url,omitempty"`
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
	UserID string `json:"user_id"`
	Role   string `json:"role"`
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
	Role string `json:"role"`
}

func (r *UpdateOrganizationMemberRequest) Validate() error {
	if strings.TrimSpace(r.Role) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type CreateOrganizationTeamRequest struct {
	Name        string          `json:"name"`
	Slug        *string         `json:"slug,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
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
	Name        string          `json:"name"`
	Slug        *string         `json:"slug,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
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
	MemberID string `json:"member_id"`
}

func (r *AddOrganizationTeamMemberRequest) Validate() error {
	if strings.TrimSpace(r.MemberID) == "" {
		return internalerrors.ErrUnprocessableEntity
	}
	return nil
}

type AcceptOrganizationInvitationRequest struct {
	RedirectURL *string `json:"redirect_url,omitempty"`
}

func (r *AcceptOrganizationInvitationRequest) Validate() error {
	if r.RedirectURL != nil {
		value := strings.TrimSpace(*r.RedirectURL)
		r.RedirectURL = &value
	}
	return nil
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
