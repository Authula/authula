package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
)

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Actor), args.Error(1)
}
