package usecases

import (
	"context"
	"errors"
	"testing"

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

	tests := []struct {
		name       string
		sessionID  *string
		user       *models.User
		userErr    error
		session    *models.Session
		sessionErr error
		want       *types.GetMeResult
		wantErr    string
	}{
		{
			name:    "returns user and most recent session when no session id is known",
			user:    expectedUser,
			session: expectedSession,
			want: &types.GetMeResult{
				User:    expectedUser,
				Session: expectedSession,
			},
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
			name:       "returns session lookup error for a known session id",
			sessionID:  new("session-123"),
			user:       expectedUser,
			sessionErr: errors.New("session lookup failed"),
			wantErr:    "session lookup failed",
		},
		{
			name:    "returns user lookup error",
			userErr: errors.New("user lookup failed"),
			session: expectedSession,
			wantErr: "user lookup failed",
		},
		{
			name:       "returns session lookup error",
			user:       expectedUser,
			sessionErr: errors.New("session lookup failed"),
			wantErr:    "session lookup failed",
		},
		{
			name:       "returns user lookup error when both fail",
			userErr:    errors.New("user lookup failed"),
			sessionErr: errors.New("session lookup failed"),
			wantErr:    "user lookup failed",
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
			} else {
				sessionService.On("GetByUserID", ctx, userID).Return(tt.session, tt.sessionErr).Once()
			}

			uc := &GetMeUseCase{
				Logger:         &inttests.MockLogger{},
				UserService:    userService,
				SessionService: sessionService,
			}

			result, err := uc.GetMe(ctx, userID, tt.sessionID)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				if result != nil {
					t.Fatalf("expected nil result on error, got %#v", result)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if result == nil {
					t.Fatal("expected result, got nil")
				} else {
					got := *result
					if got.User != tt.want.User {
						t.Fatalf("expected user %#v, got %#v", tt.want.User, got.User)
					}
					if got.Session != tt.want.Session {
						t.Fatalf("expected session %#v, got %#v", tt.want.Session, got.Session)
					}
				}
			}

			userService.AssertExpectations(t)
			sessionService.AssertExpectations(t)
		})
	}
}
