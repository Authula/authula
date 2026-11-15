package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestGetMeUseCase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := "user-123"

	expectedUser := &models.User{
		ID:    userID,
		Name:  "Test User",
		Email: "user@example.com",
	}
	expectedSession := &models.Session{
		ID:     "session-123",
		UserID: userID,
		Token:  "token-123",
	}
	otherUsersSession := &models.Session{
		ID:     "session-456",
		UserID: "someone-else",
		Token:  "token-456",
	}

	tests := []struct {
		name       string
		sessionID  *string
		user       *models.User
		userErr    error
		session    *models.Session
		sessionErr error
		want       *types.GetMeResult
		wantErr    string
		wantErrIs  error
	}{
		{
			name: "returns the user without a session for stateless credentials",
			user: expectedUser,
			want: &types.GetMeResult{User: expectedUser},
		},
		{
			name:      "returns the authenticating session when its id is known",
			sessionID: new("session-123"),
			user:      expectedUser,
			session:   expectedSession,
			want: &types.GetMeResult{
				User:    expectedUser,
				Session: expectedSession,
			},
		},
		{
			name:       "returns unauthorized when the session no longer exists",
			sessionID:  new("session-123"),
			user:       expectedUser,
			sessionErr: coreerrors.ErrSessionNotFound,
			wantErrIs:  coreerrors.ErrUnauthorized,
		},
		{
			name:      "returns unauthorized when the session lookup returns nil without an error",
			sessionID: new("session-123"),
			user:      expectedUser,
			session:   nil,
			wantErrIs: coreerrors.ErrUnauthorized,
		},
		{
			name:      "returns unauthorized when the session belongs to another user",
			sessionID: new("session-456"),
			user:      expectedUser,
			session:   otherUsersSession,
			wantErrIs: coreerrors.ErrUnauthorized,
		},
		{
			name:       "returns an unrelated session lookup error",
			sessionID:  new("session-123"),
			user:       expectedUser,
			sessionErr: errors.New("session lookup failed"),
			wantErr:    "session lookup failed",
		},
		{
			name:    "returns user lookup error",
			userErr: errors.New("user lookup failed"),
			wantErr: "user lookup failed",
		},
		{
			name:      "returns unauthorized when the user no longer exists",
			sessionID: new("session-123"),
			wantErrIs: coreerrors.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userService := &inttests.MockUserService{}
			sessionService := &inttests.MockSessionService{}

			userService.On("GetByID", ctx, userID).Return(tt.user, tt.userErr).Once()
			if tt.sessionID != nil {
				sessionService.On("GetByID", ctx, *tt.sessionID).Return(tt.session, tt.sessionErr).Once()
			}

			uc := &GetMeUseCase{
				Logger:         &inttests.MockLogger{},
				UserService:    userService,
				SessionService: sessionService,
			}

			result, err := uc.GetMe(ctx, userID, tt.sessionID)

			switch {
			case tt.wantErrIs != nil:
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Nil(t, result)
			case tt.wantErr != "":
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, result)
			default:
				require.NoError(t, err)
				require.Equal(t, tt.want, result)
			}

			userService.AssertExpectations(t)
			sessionService.AssertExpectations(t)
			if tt.sessionID == nil {
				sessionService.AssertNotCalled(t, "GetByUserID", ctx, userID)
			}
		})
	}
}
