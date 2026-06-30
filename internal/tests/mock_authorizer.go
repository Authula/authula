package tests

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/stretchr/testify/mock"
)

type NoopAuthorizer struct{}

func (a *NoopAuthorizer) AuthorizeScope(_ context.Context, _ *models.Actor, _ string) error {
	return nil
}

func (a *NoopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *models.Actor, _ string, _ string) error {
	return nil
}

type MockAuthorizer struct {
	mock.Mock
}

func (m *MockAuthorizer) AuthorizeScope(ctx context.Context, actor *models.Actor, scope string) error {
	args := m.Called(ctx, actor, scope)
	return args.Error(0)
}

func (m *MockAuthorizer) AuthorizeOrganizationAccess(ctx context.Context, actor *models.Actor, orgID string, scope string) error {
	args := m.Called(ctx, actor, orgID, scope)
	return args.Error(0)
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
