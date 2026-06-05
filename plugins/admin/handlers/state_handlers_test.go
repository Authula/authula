package handlers_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestGetUserStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name: "not found on nil state",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "user state not found",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: false}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.GetUserStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			tc.setup(userStateRepo, sessionStateRepo, impRepo)
			handler := adminhandlers.NewGetUserStateHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil, nil)
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

			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestUpsertUserStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
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
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
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

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, sessionStateRepo, impRepo)
			}
			handler := adminhandlers.NewUpsertUserStateHandler(useCase)

			request := types.UpsertUserStateRequest{Banned: true}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", body, nil)
			req.SetPathValue("user_id", "user-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
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

func TestDeleteUserStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockUserStateRepository)
		expectedStatus  int
		expectedMessage string
		expectedBody    string
	}{
		{
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("Delete", mock.Anything, "user-1").Return(internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("Delete", mock.Anything, "user-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "user state deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
			tc.setup(userStateRepo)
			handler := adminhandlers.NewDeleteUserStateHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/users/user-1", nil, nil)
			req.SetPathValue("user_id", "user-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectedBody != "" {
				payload := internaltests.DecodeResponseJSON[types.DeleteUserStateResponse](t, reqCtx)
				if payload.Message != tc.expectedBody {
					t.Fatalf("expected %s, got %s", tc.expectedBody, payload.Message)
				}
			}

			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestGetBannedUserStatesHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockUserStateRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("GetBanned", mock.Anything).Return(([]types.AdminUserState)(nil), errors.New("internal error")).Once()
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "internal error",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository) {
				userStateRepo.On("GetBanned", mock.Anything).Return([]types.AdminUserState{{UserID: "user-1", Banned: true}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[[]types.AdminUserState](t, reqCtx)
				if payload == nil || len(payload) != 1 {
					t.Fatalf("expected banned user state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
			tc.setup(userStateRepo)
			handler := adminhandlers.NewGetBannedUserStatesHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/banned", nil, nil)
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

			userStateRepo.AssertExpectations(t)
		})
	}
}

func TestBanUserHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
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
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, _ *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.BanUserResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, sessionStateRepo, impRepo)
			}
			handler := adminhandlers.NewBanUserHandler(useCase)

			request := types.BanUserRequest{}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", body, nil)
			req.SetPathValue("user_id", "user-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
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

func TestUnbanUserHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockUserStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(userStateRepo *admintests.MockUserStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: false}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UnbanUserResponse](t, reqCtx)
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
			tc.setup(userStateRepo, impRepo)
			handler := adminhandlers.NewUnbanUserHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/unban", nil, nil)
			req.SetPathValue("user_id", "user-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
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

func TestGetSessionStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name: "not found on nil",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "session state not found",
		},
		{
			name: "success",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.GetSessionStateResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			tc.setup(userStateRepo, sessionStateRepo, impRepo)
			handler := adminhandlers.NewGetSessionStateHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil, nil)
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

func TestUpsertSessionStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
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
			name: "error",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name: "success",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
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

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, sessionStateRepo, impRepo)
			}
			handler := adminhandlers.NewUpsertSessionStateHandler(useCase)

			request := types.UpsertSessionStateRequest{Revoke: true}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", body, nil)
			req.SetPathValue("session_id", "session-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
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

func TestDeleteSessionStateHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockSessionStateRepository)
		expectedStatus  int
		expectedMessage string
		expectedBody    string
	}{
		{
			name: "error",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("Delete", mock.Anything, "session-1").Return(internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("Delete", mock.Anything, "session-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "session state deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			tc.setup(sessionStateRepo)
			handler := adminhandlers.NewDeleteSessionStateHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/sessions/session-1", nil, nil)
			req.SetPathValue("session_id", "session-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectedBody != "" {
				payload := internaltests.DecodeResponseJSON[types.DeleteSessionStateResponse](t, reqCtx)
				if payload.Message != tc.expectedBody {
					t.Fatalf("expected %s, got %s", tc.expectedBody, payload.Message)
				}
			}

			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestGetRevokedSessionStatesHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockSessionStateRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("GetRevoked", mock.Anything).Return(([]types.AdminSessionState)(nil), errors.New("internal error")).Once()
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "internal error",
		},
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository) {
				sessionStateRepo.On("GetRevoked", mock.Anything).Return([]types.AdminSessionState{{SessionID: "session-1"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[[]types.AdminSessionState](t, reqCtx)
				if payload == nil || len(payload) != 1 {
					t.Fatalf("expected session state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
			tc.setup(sessionStateRepo)
			handler := adminhandlers.NewGetRevokedSessionStatesHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/revoked", nil, nil)
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

func TestGetUserAdminSessionsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
		expectedStatus  int
		expectedMessage string
		assertResponse  func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "error",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				sessionStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(([]types.AdminUserSession)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(sessionStateRepo *admintests.MockSessionStateRepository, impRepo *admintests.MockImpersonationRepository) {
				expiresAt := time.Now().UTC().Add(time.Hour)
				impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
				sessionStateRepo.On("GetByUserID", mock.Anything, "user-1").Return([]types.AdminUserSession{{Session: models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: expiresAt}}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[[]types.AdminUserSession](t, reqCtx)
				if payload == nil || len(payload) != 1 {
					t.Fatalf("expected user sessions, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			tc.setup(sessionStateRepo, impRepo)
			handler := adminhandlers.NewGetUserAdminSessionsHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1/sessions", nil, nil)
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
			sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestRevokeSessionHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*admintests.MockUserStateRepository, *admintests.MockSessionStateRepository, *admintests.MockImpersonationRepository)
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
			name: "use case error",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name: "success",
			setup: func(_ *admintests.MockUserStateRepository, sessionStateRepo *admintests.MockSessionStateRepository, _ *admintests.MockImpersonationRepository) {
				reason := "suspicious"
				sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
				sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
				sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1", RevokedReason: &reason}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			assertResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.RevokeSessionResponse](t, reqCtx)
				if payload.State == nil {
					t.Fatalf("expected session state, got %v", payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, userStateRepo, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
			if tc.setup != nil {
				tc.setup(userStateRepo, sessionStateRepo, impRepo)
			}
			handler := adminhandlers.NewRevokeSessionHandler(useCase)

			reason := "security"
			request := types.RevokeSessionRequest{Reason: &reason}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", body, nil)
			req.SetPathValue("session_id", "session-1")
			actorID := "actor-1"
			reqCtx.Actor = &models.Actor{ID: actorID, Type: models.ActorUser}
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
