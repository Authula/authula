package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	rootservices "github.com/Authula/authula/services"
)

type MockRateLimitService struct {
	mock.Mock
}

func (m *MockRateLimitService) GetValue(ctx context.Context, key string) (any, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *MockRateLimitService) CheckAndIncrement(ctx context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	args := m.Called(ctx, key, window, maxRequests)
	return args.Bool(0), args.Int(1), args.Get(2).(time.Time), args.Error(3)
}

func (m *MockRateLimitService) SetRule(ctx context.Context, key string, window time.Duration, maxRequests int) error {
	args := m.Called(ctx, key, window, maxRequests)
	return args.Error(0)
}

func (m *MockRateLimitService) GetRule(ctx context.Context, key string) (*rootservices.RateLimitKeyRule, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rootservices.RateLimitKeyRule), args.Error(1)
}

func (m *MockRateLimitService) DeleteRule(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type MockOrganizationService struct {
	mock.Mock
}

func (m *MockOrganizationService) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	args := m.Called(ctx, organizationID)
	return args.Bool(0), args.Error(1)
}

func (m *MockOrganizationService) GetUserPermissionsInOrganization(ctx context.Context, userID string, organizationID string) ([]string, error) {
	args := m.Called(ctx, userID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
