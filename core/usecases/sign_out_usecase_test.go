package usecases

import (
	"context"
	"errors"
	"testing"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/types"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestSignOutUseCase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := "user-123"

	const (
		currentSessionID   = "session-current"
		requestedSessionID = "session-requested"
	)

	tests := []struct {
		name               string
		currentSessionID   *string
		requestedSessionID *string
		signOutAll         *bool
		configure          func(*inttests.MockSessionService)
		want               *types.SignOutResult
		wantErr            error
	}{
		{
			name:       "deletes all sessions when sign out all is requested",
			signOutAll: new(true),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", ctx, userID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out from all sessions", ClearedCurrentSession: true},
		},
		{
			name:       "returns delete all error when deleting all sessions fails",
			signOutAll: new(true),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", ctx, userID).Return(errors.New("delete all failed")).Once()
			},
			wantErr: errors.New("delete all failed"),
		},
		{
			name:               "sign out all takes precedence over a requested session id",
			currentSessionID:   new(currentSessionID),
			requestedSessionID: new(requestedSessionID),
			signOutAll:         new(true),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", ctx, userID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out from all sessions", ClearedCurrentSession: true},
		},
		{
			name:               "deletes a requested session owned by the user without clearing the current session",
			currentSessionID:   new(currentSessionID),
			requestedSessionID: new(requestedSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, requestedSessionID).Return(&models.Session{ID: requestedSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", ctx, requestedSessionID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out", ClearedCurrentSession: false},
		},
		{
			name:               "clears the current session when the requested session id is the current session",
			currentSessionID:   new(currentSessionID),
			requestedSessionID: new(currentSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, currentSessionID).Return(&models.Session{ID: currentSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", ctx, currentSessionID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out", ClearedCurrentSession: true},
		},
		{
			name:               "returns not found when the requested session does not exist",
			currentSessionID:   new(currentSessionID),
			requestedSessionID: new(requestedSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, requestedSessionID).Return((*models.Session)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:               "returns forbidden when the requested session belongs to another user",
			currentSessionID:   new(currentSessionID),
			requestedSessionID: new(requestedSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, requestedSessionID).Return(&models.Session{ID: requestedSessionID, UserID: "other-user"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:               "returns get by id error when loading the requested session fails",
			requestedSessionID: new(requestedSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, requestedSessionID).Return((*models.Session)(nil), errors.New("get by id failed")).Once()
			},
			wantErr: errors.New("get by id failed"),
		},
		{
			name:               "returns delete error when deleting the requested session fails",
			requestedSessionID: new(requestedSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("GetByID", ctx, requestedSessionID).Return(&models.Session{ID: requestedSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", ctx, requestedSessionID).Return(errors.New("delete failed")).Once()
			},
			wantErr: errors.New("delete failed"),
		},
		{
			name:             "deletes the current session when no explicit sign out target is provided",
			currentSessionID: new(currentSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("Delete", ctx, currentSessionID).Return(nil).Once()
			},
			want: &types.SignOutResult{Message: "signed out", ClearedCurrentSession: true},
		},
		{
			name:             "returns delete error when deleting the current session fails",
			currentSessionID: new(currentSessionID),
			configure: func(sessionService *inttests.MockSessionService) {
				sessionService.On("Delete", ctx, currentSessionID).Return(errors.New("delete current failed")).Once()
			},
			wantErr: errors.New("delete current failed"),
		},
		{
			name:    "returns an error when there is no current session and no sign out target",
			wantErr: coreerrors.ErrNoSessionToSignOut,
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

			result, err := uc.SignOut(ctx, userID, tt.currentSessionID, tt.requestedSessionID, tt.signOutAll)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
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
				}
				if *result != *tt.want {
					t.Fatalf("expected result %#v, got %#v", *tt.want, *result)
				}
			}

			sessionService.AssertExpectations(t)
		})
	}
}
