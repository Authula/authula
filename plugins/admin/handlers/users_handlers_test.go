package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestCreateUserHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
		expectUser      bool
	}{
		{
			name:            "invalid request body",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "use case error",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "existing", Email: "user@example.com"}, nil).Once()
			},
			expectedStatus:  http.StatusConflict,
			expectedMessage: "conflict",
		},
		{
			name: "success",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").Return((*models.User)(nil), nil).Once()
				repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{
					ID:    "user-1",
					Name:  "User",
					Email: "user@example.com",
				}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectUser:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			handler := adminhandlers.NewCreateUserHandler(useCase)
			if tc.setup != nil {
				tc.setup(repo)
			}

			request := types.CreateUserRequest{Name: "User", Email: "user@example.com"}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users", body, nil)
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectUser {
				payload := internaltests.DecodeResponseJSON[types.CreateUserResponse](t, reqCtx)
				if payload.User == nil {
					t.Fatalf("expected user in payload, got %v", payload)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestGetAllUsersHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		setup           func(*internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
		assertPage      func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:            "invalid limit",
			path:            "/admin/users?limit=invalid",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid limit",
		},
		{
			name: "use case error",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetAll", mock.Anything, (*string)(nil), 10).Return(([]models.User)(nil), (*string)(nil), internalerrors.ErrForbidden).Once()
			},
			path:            "/admin/users",
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name: "success with cursor and limit",
			setup: func(repo *internaltests.MockUserRepository) {
				cursor := "next-cursor"
				queryCursor := "cur-1"
				repo.On("GetAll", mock.Anything, &queryCursor, 5).Return([]models.User{{ID: "user-1", Email: "u1@example.com"}}, &cursor, nil).Once()
			},
			path:           "/admin/users?cursor=cur-1&limit=5",
			expectedStatus: http.StatusOK,
			assertPage: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UsersPage](t, reqCtx)
				if payload.Users == nil {
					t.Fatalf("expected users field in payload, got %v", payload)
				}
				if len(payload.Users) != 1 {
					t.Fatalf("expected 1 user, got %d", len(payload.Users))
				}
				if payload.NextCursor == nil || *payload.NextCursor != "next-cursor" {
					t.Fatalf("expected next cursor to be next-cursor, got %v", payload.NextCursor)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			handler := adminhandlers.NewGetAllUsersHandler(useCase)
			if tc.setup != nil {
				tc.setup(repo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tc.path, nil, nil)
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.assertPage != nil {
				tc.assertPage(t, reqCtx)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestGetUserByIDHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pathParams      map[string]string
		setup           func(*internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
		expectUser      bool
	}{
		{
			name:       "use case error",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), internalerrors.ErrUnauthorized).Once()
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
		},
		{
			name:       "not found",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "user not found",
		},
		{
			name:       "success",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			handler := adminhandlers.NewGetUserByIDHandler(useCase)
			if tc.setup != nil {
				tc.setup(repo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil, nil)
			for k, v := range tc.pathParams {
				req.SetPathValue(k, v)
			}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectUser {
				payload := internaltests.DecodeResponseJSON[types.GetUserByIDResponse](t, reqCtx)
				if payload.User == nil {
					t.Fatalf("expected user in payload, got %v", payload)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestUpdateUserHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		pathParams      map[string]string
		setup           func(*internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
		expectUser      bool
	}{
		{
			name:            "invalid request body",
			body:            []byte("{invalid"),
			pathParams:      map[string]string{"user_id": "user-1"},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name:       "use case error",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name:       "success",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Name: "Old"}, nil).Once()
				repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{ID: "user-1", Name: "Updated"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectUser:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			handler := adminhandlers.NewUpdateUserHandler(useCase)
			if tc.setup != nil {
				tc.setup(repo)
			}

			request := types.UpdateUserRequest{Name: admintests.PtrString(t, "Updated")}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", body, nil)
			for k, v := range tc.pathParams {
				req.SetPathValue(k, v)
			}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectUser {
				payload := internaltests.DecodeResponseJSON[types.UpdateUserResponse](t, reqCtx)
				if payload.User == nil {
					t.Fatalf("expected user in payload, got %v", payload)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestDeleteUserHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pathParams      map[string]string
		setup           func(*internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
		expectedBody    string
	}{
		{
			name:       "use case error",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
				repo.On("Delete", mock.Anything, "user-1").Return(internalerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name:       "success",
			pathParams: map[string]string{"user_id": "user-1"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
				repo.On("Delete", mock.Anything, "user-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "user deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			handler := adminhandlers.NewDeleteUserHandler(useCase)
			if tc.setup != nil {
				tc.setup(repo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/users/user-1", nil, nil)
			for k, v := range tc.pathParams {
				req.SetPathValue(k, v)
			}
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectedBody != "" {
				payload := internaltests.DecodeResponseJSON[types.DeleteUserResponse](t, reqCtx)
				if payload.Message != tc.expectedBody {
					t.Fatalf("expected %s, got %s", tc.expectedBody, payload.Message)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}
