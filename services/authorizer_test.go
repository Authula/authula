package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

func TestNewDefaultAuthorizer(t *testing.T) {
	t.Parallel()

	authorizer := NewDefaultAuthorizer()

	require.NotNil(t, authorizer)
	require.Implements(t, (*Authorizer)(nil), authorizer)
}

func TestDefaultAuthorizerAuthorizeScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actor   *models.Actor
		scope   string
		wantErr error
	}{
		{
			name:    "rejects a nil actor",
			actor:   nil,
			scope:   "users:read",
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:    "rejects an actor without an ID",
			actor:   &models.Actor{Type: models.ActorUser, Scopes: []string{"*"}},
			scope:   "users:read",
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:    "rejects an actor without scopes",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser},
			scope:   "users:read",
			wantErr: coreerrors.ErrInsufficientPermissions,
		},
		{
			name:    "grants the wildcard scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"*"}},
			scope:   "users:read",
			wantErr: nil,
		},
		{
			name:    "grants an exact scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"users:read"}},
			scope:   "users:read",
			wantErr: nil,
		},
		{
			name:    "grants a scope matched by a prefix wildcard",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"users:*"}},
			scope:   "users:read",
			wantErr: nil,
		},
		{
			name:    "grants a scope matched by a bare prefix wildcard",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"users*"}},
			scope:   "users:read",
			wantErr: nil,
		},
		{
			name:    "rejects a prefix wildcard that does not match the namespace",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"users:*"}},
			scope:   "users",
			wantErr: coreerrors.ErrInsufficientPermissions,
		},
		{
			name:    "rejects an unrelated scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"accounts:read"}},
			scope:   "users:read",
			wantErr: coreerrors.ErrInsufficientPermissions,
		},
		{
			name:    "rejects an empty required scope",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"users:read"}},
			scope:   "",
			wantErr: coreerrors.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := NewDefaultAuthorizer()

			err := authorizer.AuthorizeScope(context.Background(), tt.actor, tt.scope)

			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
		})
	}
}

func TestDefaultAuthorizerAuthorizeOrganizationAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actor   *models.Actor
		orgID   string
		wantErr error
	}{
		{
			name:    "rejects a nil actor",
			actor:   nil,
			orgID:   "org-1",
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:    "rejects an actor without an ID",
			actor:   &models.Actor{Type: models.ActorUser, Claims: map[string]any{"organization_id": "org-1"}},
			orgID:   "org-1",
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:    "allows an empty target organization",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:   "",
			wantErr: nil,
		},
		{
			name:    "allows an actor without an organization claim",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:   "org-1",
			wantErr: nil,
		},
		{
			name:    "allows an actor with an empty organization claim",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Claims: map[string]any{"organization_id": ""}},
			orgID:   "org-1",
			wantErr: nil,
		},
		{
			name:    "allows an actor bound to the target organization",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Claims: map[string]any{"organization_id": "org-1"}},
			orgID:   "org-1",
			wantErr: nil,
		},
		{
			name:    "rejects an actor bound to a different organization",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Claims: map[string]any{"organization_id": "org-2"}},
			orgID:   "org-1",
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:    "ignores a non-string organization claim",
			actor:   &models.Actor{ID: "user-1", Type: models.ActorUser, Claims: map[string]any{"organization_id": 42}},
			orgID:   "org-1",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := NewDefaultAuthorizer()

			err := authorizer.AuthorizeOrganizationAccess(context.Background(), tt.actor, tt.orgID)

			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
		})
	}
}
