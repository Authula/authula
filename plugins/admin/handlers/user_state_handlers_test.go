package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
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
				userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), coreerrors.ErrBadRequest).Once()
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
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(coreerrors.ErrBadRequest).Once()
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
				userStateRepo.On("Delete", mock.Anything, "user-1").Return(coreerrors.ErrNotFound).Once()
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
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(coreerrors.ErrBadRequest).Once()
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
				userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(coreerrors.ErrNotFound).Once()
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
