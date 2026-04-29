package tests

import (
	"context"

	"github.com/Authula/authula/plugins/magic-link/types"
	"github.com/stretchr/testify/mock"
)

// Usecases

type MockVerifyUseCase struct {
	mock.Mock
}

func (m *MockVerifyUseCase) Verify(ctx context.Context, token string, ipAddress *string, userAgent *string) (string, error) {
	args := m.Called(ctx, token, ipAddress, userAgent)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

type MockExchangeUseCase struct {
	mock.Mock
}

func (m *MockExchangeUseCase) Exchange(ctx context.Context, token string, ipAddress *string, userAgent *string) (*types.ExchangeResult, error) {
	args := m.Called(ctx, token, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ExchangeResult), args.Error(1)
}
