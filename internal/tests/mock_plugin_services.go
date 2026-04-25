package tests

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockOrganizationService struct {
	mock.Mock
}

func (m *MockOrganizationService) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	args := m.Called(ctx, organizationID)
	return args.Bool(0), args.Error(1)
}
