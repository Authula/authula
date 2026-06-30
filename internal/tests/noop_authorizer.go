package tests

import (
	"context"

	"github.com/Authula/authula/models"
)

type NoopAuthorizer struct{}

func (a *NoopAuthorizer) AuthorizeScope(_ context.Context, _ *models.Actor, _ string) error {
	return nil
}

func (a *NoopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *models.Actor, _ string, _ string) error {
	return nil
}

func TestActor() *models.Actor {
	return &models.Actor{ID: "test-actor", Type: models.ActorUser, Scopes: []string{"*"}}
}

func ActorFromUserID(userID *string) *models.Actor {
	if userID == nil {
		return nil
	}
	return &models.Actor{ID: *userID, Type: models.ActorUser, Scopes: []string{"*"}}
}
