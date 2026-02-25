package service_test

import (
	"context"
	"testing"

	"yourproject/internal/domain"
	"yourproject/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetByID_ReturnsUser_WhenExists(t *testing.T) {
	t.Parallel()

	// Arrange
	mockRepo := &MockUserRepository{}
	expectedUser := &domain.User{ID: 1, Name: "Alice"}
	mockRepo.On("GetByID", mock.Anything, int64(1)).Return(expectedUser, nil)

	svc := service.NewUserService(mockRepo)
	ctx := context.Background()

	// Act
	user, err := svc.GetByID(ctx, 1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetByID_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      int64
		setup   func(*MockUserRepository)
		wantErr bool
	}{
		{
			name: "user exists",
			id:   1,
			setup: func(m *MockUserRepository) {
				m.On("GetByID", mock.Anything, int64(1)).Return(&domain.User{ID: 1}, nil)
			},
		},
		{
			name: "user not found",
			id:   999,
			setup: func(m *MockUserRepository) {
				m.On("GetByID", mock.Anything, int64(999)).Return(nil, domain.ErrUserNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockUserRepository{}
			tt.setup(mockRepo)
			svc := service.NewUserService(mockRepo)
			ctx := context.Background()

			// Act
			_, err := svc.GetByID(ctx, tt.id)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
