package services

import (
	"context"
	"strings"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

type Authorizer interface {
	AuthorizeScope(ctx context.Context, actor *models.Actor, scope string) error
	AuthorizeOrganizationAccess(ctx context.Context, actor *models.Actor, orgID string, scope string) error
}

type DefaultAuthorizer struct{}

func NewDefaultAuthorizer() Authorizer {
	return &DefaultAuthorizer{}
}

func (a *DefaultAuthorizer) AuthorizeScope(ctx context.Context, actor *models.Actor, scope string) error {
	if actor == nil || actor.ID == "" {
		return coreerrors.ErrUnauthorized
	}
	if !hasPermission(actor.Scopes, scope) {
		return coreerrors.ErrInsufficientPermissions
	}
	return nil
}

func (a *DefaultAuthorizer) AuthorizeOrganizationAccess(ctx context.Context, actor *models.Actor, orgID string, scope string) error {
	if actor == nil || actor.ID == "" {
		return coreerrors.ErrUnauthorized
	}
	if err := verifyTenant(actor, orgID); err != nil {
		return err
	}
	if !hasPermission(actor.Scopes, scope) {
		return coreerrors.ErrInsufficientPermissions
	}
	return nil
}

func verifyTenant(actor *models.Actor, targetOrgID string) error {
	if targetOrgID == "" {
		return nil
	}
	boundOrg, ok := actor.GetClaimString("organization_id")
	if !ok || boundOrg == "" {
		return nil
	}
	if boundOrg != targetOrgID {
		return coreerrors.ErrForbidden
	}
	return nil
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
