package usecases

import (
	"context"
	"errors"
	"testing"

	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/internal/types"
	"github.com/Authula/authula/models"
)

func TestSignOutUseCase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := "user-123"

	const (
		sessionIDValue  = "session-123"
		recentSessionID = "session-456"
	)

	tests := []struct {
		name               string
		sessionID          string
		sessionIDProvided  bool
		signOutAll         bool
		signOutAllProvided bool
		configure          func(*inttests.MockSessionService)
		want               *types.SignOutResult
		wantErr            string
	}{
		{
			name:              "deletes a specific session when session id is provided",
			sessionID:         sessionIDValue,
			sessionIDProvided: true,
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("Delete", ctx, sessionIDValue).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out"},
		},
		{
			name:              "returns delete session error when deleting a specific session fails",
			sessionID:         sessionIDValue,
			sessionIDProvided: true,
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("Delete", ctx, sessionIDValue).Return(errors.New("delete failed")).Once()
			},
			wantErr: "delete failed",
		},
		{
			name:               "deletes all sessions when sign out all is requested",
			signOutAll:         true,
			signOutAllProvided: true,
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", ctx, userID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out from all sessions"},
		},
		{
			name:               "returns delete all error when deleting all sessions fails",
			signOutAll:         true,
			signOutAllProvided: true,
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", ctx, userID).Return(errors.New("delete all failed")).Once()
			},
			wantErr: "delete all failed",
		},
		{
			name: "deletes the most recent session when no explicit sign out target is provided",
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByUserID", ctx, userID).Return(&models.Session{ID: recentSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", ctx, recentSessionID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out"},
		},
		{
			name: "returns get session error when loading the most recent session fails",
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByUserID", ctx, userID).Return((*models.Session)(nil), errors.New("get failed")).Once()
			},
			wantErr: "get failed",
		},
		{
			name: "returns no active session message when the user has no session",
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByUserID", ctx, userID).Return((*models.Session)(nil), nil).Once()
			},
			want: &types.SignOutResult{Message: "no active session found"},
		},
		{
			name: "returns delete error when deleting the most recent session fails",
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByUserID", ctx, userID).Return(&models.Session{ID: recentSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", ctx, recentSessionID).Return(errors.New("delete recent failed")).Once()
			},
			wantErr: "delete recent failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessionService := &inttests.MockSessionService{}
			if tt.configure != nil {
				tt.configure(sessionService)
			}

			uc := &SignOutUseCase{
				Logger:         &inttests.MockLogger{},
				SessionService: sessionService,
			}

			var sessionID *string
			if tt.sessionIDProvided {
				value := tt.sessionID
				sessionID = &value
			}

			var signOutAll *bool
			if tt.signOutAllProvided {
				value := tt.signOutAll
				signOutAll = &value
			}

			result, err := uc.SignOut(ctx, userID, sessionID, signOutAll)

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
					if got.Message != tt.want.Message {
						t.Fatalf("expected message %q, got %q", tt.want.Message, got.Message)
					}
				}
			}

			sessionService.AssertExpectations(t)
		})
	}
}
