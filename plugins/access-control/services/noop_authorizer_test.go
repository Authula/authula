package services

import (
	"context"

	"github.com/Authula/authula/models"
)

type noopAuthorizer struct{}

func (a *noopAuthorizer) AuthorizeScope(_ context.Context, _ *models.Actor, _ string) error {
	return nil
}

func (a *noopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *models.Actor, _ string, _ string) error {
	return nil
}

func testActor() *models.Actor {
	return &models.Actor{ID: "test-actor", Scopes: []string{"*"}}
}
