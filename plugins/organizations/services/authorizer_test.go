package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

func authorizer() rootservices.Authorizer {
	return rootservices.NewDefaultAuthorizer()
}

func TestAuthorizer_AuthorizeScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		actor   *models.Actor
		scope   string
		wantErr error
	}{
		{
			name:    "nil actor returns unauthorized",
			actor:   nil,
			scope:   "organizations:create",
			wantErr: internalerrors.ErrUnauthorized,
		},
		{
			name:    "empty ID actor returns unauthorized",
			actor:   &models.Actor{ID: "", Type: models.ActorUser},
			scope:   "organizations:create",
			wantErr: internalerrors.ErrUnauthorized,
		},
		{
			name:  "allows actor with matching scope",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"organizations:create"}},
			scope: "organizations:create",
		},
		{
			name:    "denies actor without matching scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser},
			scope:   "organizations:create",
			wantErr: internalerrors.ErrInsufficientPermissions,
		},
		{
			name:  "allows wildcard scope",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"*"}},
			scope: "organizations:create",
		},
		{
			name:  "allows prefix wildcard scope",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"organizations:*"}},
			scope: "organizations:create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := authorizer()
			err := a.AuthorizeScope(ctx, tt.actor, tt.scope)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthorizer_AuthorizeOrganizationAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		actor   *models.Actor
		orgID   string
		scope   string
		wantErr error
	}{
		{
			name:    "nil actor returns unauthorized",
			actor:   nil,
			orgID:   "org-1",
			scope:   "organizations:read",
			wantErr: internalerrors.ErrUnauthorized,
		},
		{
			name:    "empty ID actor returns unauthorized",
			actor:   &models.Actor{ID: "", Type: models.ActorUser},
			orgID:   "org-1",
			scope:   "organizations:read",
			wantErr: internalerrors.ErrUnauthorized,
		},
		{
			name:  "allows user with matching scope",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"organizations:read"}},
			orgID: "org-1",
			scope: "organizations:read",
		},
		{
			name:  "allows machine with matching scope and org claim",
			actor: &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"organizations:read"}},
			orgID: "org-1",
			scope: "organizations:read",
		},
		{
			name:    "denies machine with wrong org claim",
			actor:   &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"organizations:read"}},
			orgID:   "org-2",
			scope:   "organizations:read",
			wantErr: internalerrors.ErrForbidden,
		},
		{
			name:    "denies actor without scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:   "org-1",
			scope:   "organizations:read",
			wantErr: internalerrors.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := authorizer()
			err := a.AuthorizeOrganizationAccess(ctx, tt.actor, tt.orgID, tt.scope)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
