package services

import (
	"context"
	"testing"

	"github.com/Authula/authula/models"
)

func TestDefaultAuthorizerAuthorize(t *testing.T) {
	tests := []struct {
		name     string
		actor    *models.Actor
		action   AuthorizerAction
		resource AuthorizerResource
		wantErr  bool
	}{
		{
			name:     "allows user actors without permissions",
			actor:    &models.Actor{ID: "user-1", Type: models.ActorUser, Claims: map[string]any{"organization_id": "org-1"}},
			action:   ActionOrganizationsDelete,
			resource: AuthorizerResource{OrganizationID: "org-1"},
			wantErr:  false,
		},
		{
			name:     "denies machine actors for list actions",
			actor:    &models.Actor{ID: "machine-1", Type: models.ActorMachine, Scopes: []string{"organizations:list"}},
			action:   ActionOrganizationsList,
			resource: AuthorizerResource{OrganizationID: "org-1"},
			wantErr:  true,
		},
		{
			name:     "allows machine actors with matching permission and organization",
			actor:    &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"organizations:teams:create"}},
			action:   ActionOrganizationsTeamsCreate,
			resource: AuthorizerResource{OrganizationID: "org-1"},
			wantErr:  false,
		},
		{
			name:     "denies machine actors without matching organization",
			actor:    &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-2"}, Scopes: []string{"organizations:teams:create"}},
			action:   ActionOrganizationsTeamsCreate,
			resource: AuthorizerResource{OrganizationID: "org-1"},
			wantErr:  true,
		},
		{
			name:     "allows wildcard permissions for machine actors",
			actor:    &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"organizations:*"}},
			action:   ActionOrganizationsMembersRemove,
			resource: AuthorizerResource{OrganizationID: "org-1"},
			wantErr:  false,
		},
	}

	authorizer := NewAuthorizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.Authorize(context.Background(), tt.actor, tt.action, tt.resource)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
