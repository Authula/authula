package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestCreateUserStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "invalid json",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "service error",
			setup: func(_ *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(false, nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), nil).Once()
				userStateRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UpsertUserStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, impRepo)
			}
			handler := adminhandlers.NewCreateUserStateHandler(useCase)

			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, types.CreateUserStateRequest{Banned: true})
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users/user-1/state", body, nil)
			req.SetPathValue("user_id", "user-1")
			reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.assertResponse != nil {
				tc.assertResponse(t, reqCtx)
			}

			impRepo.AssertExpectations(t)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateUserStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "invalid json",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "state not found",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "service error",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
				userStateRepo.On("Update", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(errors.New("internal error")).Once()
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "internal error",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
				userStateRepo.On("Update", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UpsertUserStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, impRepo)
			}
			handler := adminhandlers.NewUpdateUserStateHandler(useCase)

			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, types.UpsertUserStateRequest{Banned: true})
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/users/user-1/state", body, nil)
			req.SetPathValue("user_id", "user-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.assertResponse != nil {
				tc.assertResponse(t, reqCtx)
			}

			impRepo.AssertExpectations(t)
			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestCreateSessionStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockSessionStateRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "invalid json",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "service error",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "conflict",
		},
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), nil).Once()
				sessionStateRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UpsertSessionStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(sessionStateRepo)
			}
			handler := adminhandlers.NewCreateSessionStateHandler(useCase)

			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, types.CreateSessionStateRequest{Revoke: true})
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/sessions/session-1/state", body, nil)
			req.SetPathValue("session_id", "session-1")
			reqCtx.Actor = &models.Actor{ID: "actor-1", Type: models.ActorUser}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.assertResponse != nil {
				tc.assertResponse(t, reqCtx)
			}

			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateSessionStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockSessionStateRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "invalid json",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "state not found",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
				sessionStateRepo.On("Update", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UpsertSessionStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(sessionStateRepo)
			}
			handler := adminhandlers.NewUpdateSessionStateHandler(useCase)

			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, types.UpsertSessionStateRequest{Revoke: true})
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/sessions/session-1/state", body, nil)
			req.SetPathValue("session_id", "session-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.assertResponse != nil {
				tc.assertResponse(t, reqCtx)
			}

			sessionStateRepo.AssertExpectations(t)
		})
	}
}
