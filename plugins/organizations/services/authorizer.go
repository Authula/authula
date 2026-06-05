package services

import (
	"context"
	"strings"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
)

type AuthorizerAction string

const (
	ActionOrganizationsList              AuthorizerAction = "organizations:list"
	ActionOrganizationsCreate            AuthorizerAction = "organizations:create"
	ActionOrganizationsRead              AuthorizerAction = "organizations:read"
	ActionOrganizationsUpdate            AuthorizerAction = "organizations:update"
	ActionOrganizationsDelete            AuthorizerAction = "organizations:delete"
	ActionOrganizationsInvitationsList   AuthorizerAction = "organizations:invitations:list"
	ActionOrganizationsInvitationsCreate AuthorizerAction = "organizations:invitations:create"
	ActionOrganizationsInvitationsRead   AuthorizerAction = "organizations:invitations:read"
	ActionOrganizationsInvitationsRevoke AuthorizerAction = "organizations:invitations:revoke"
	ActionOrganizationsInvitationsAccept AuthorizerAction = "organizations:invitations:accept"
	ActionOrganizationsInvitationsReject AuthorizerAction = "organizations:invitations:reject"
	ActionOrganizationsMembersList       AuthorizerAction = "organizations:members:list"
	ActionOrganizationsMembersAdd        AuthorizerAction = "organizations:members:add"
	ActionOrganizationsMembersRead       AuthorizerAction = "organizations:members:read"
	ActionOrganizationsMembersUpdate     AuthorizerAction = "organizations:members:update"
	ActionOrganizationsMembersRemove     AuthorizerAction = "organizations:members:remove"
	ActionOrganizationsTeamsList         AuthorizerAction = "organizations:teams:list"
	ActionOrganizationsTeamsCreate       AuthorizerAction = "organizations:teams:create"
	ActionOrganizationsTeamsRead         AuthorizerAction = "organizations:teams:read"
	ActionOrganizationsTeamsUpdate       AuthorizerAction = "organizations:teams:update"
	ActionOrganizationsTeamsDelete       AuthorizerAction = "organizations:teams:delete"
	ActionOrganizationsTeamMembersList   AuthorizerAction = "organizations:team-members:list"
	ActionOrganizationsTeamMembersAdd    AuthorizerAction = "organizations:team-members:add"
	ActionOrganizationsTeamMembersRead   AuthorizerAction = "organizations:team-members:read"
	ActionOrganizationsTeamMembersRemove AuthorizerAction = "organizations:team-members:remove"
)

type AuthorizerResource struct {
	OrganizationID string
}

type Authorizer interface {
	Authorize(ctx context.Context, actor *models.Actor, action AuthorizerAction, resource AuthorizerResource) error
}

type DefaultAuthorizer struct{}

func NewAuthorizer() Authorizer {
	return &DefaultAuthorizer{}
}

func NewDefaultAuthorizer() Authorizer {
	return NewAuthorizer()
}

func (a *DefaultAuthorizer) Authorize(ctx context.Context, actor *models.Actor, action AuthorizerAction, resource AuthorizerResource) error {
	if actor == nil || actor.ID == "" {
		return internalerrors.ErrUnauthorized
	}

	if actor.Type != models.ActorUser && actor.Type != models.ActorMachine {
		return internalerrors.ErrForbidden
	}

	if actor.Type == models.ActorMachine && isUserOnlyAction(action) {
		return internalerrors.ErrForbidden
	}

	if resource.OrganizationID != "" {
		if actor.OrganizationID == nil || *actor.OrganizationID == "" {
			return internalerrors.ErrForbidden
		}
		if *actor.OrganizationID != resource.OrganizationID {
			return internalerrors.ErrForbidden
		}
	}

	requiredPermission := string(action)
	if requiredPermission == "" {
		return internalerrors.ErrForbidden
	}

	if actor.Type == models.ActorMachine {
		if !hasPermission(actor.Scopes, requiredPermission) {
			return internalerrors.ErrForbidden
		}
		return nil
	}

	return nil
}

func isUserOnlyAction(action AuthorizerAction) bool {
	switch action {
	case ActionOrganizationsCreate,
		ActionOrganizationsDelete,
		ActionOrganizationsList:
		return true

	case ActionOrganizationsInvitationsAccept,
		ActionOrganizationsInvitationsReject,
		ActionOrganizationsInvitationsCreate,
		ActionOrganizationsInvitationsRevoke:
		return true

	default:
		return false
	}
}

func hasPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == required {
			return true
		}
		if before, ok := strings.CutSuffix(permission, "*"); ok {
			prefix := before
			if strings.HasPrefix(required, prefix) {
				return true
			}
		}
	}

	return false
}
