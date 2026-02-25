package service_test

import (
	"context"
	"yourproject/models"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository implements service.UserRepository for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}
