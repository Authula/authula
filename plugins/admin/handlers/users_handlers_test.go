package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestCreateUserHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		handler := adminhandlers.NewCreateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users", []byte("{invalid"))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		request := types.CreateUserRequest{Name: "User", Email: "user@example.com"}
		repo.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "existing", Email: "user@example.com"}, nil).Once()
		handler := adminhandlers.NewCreateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users", internaltests.MarshalToJSON(t, request))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusConflict, "conflict")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		request := types.CreateUserRequest{Name: "User", Email: "user@example.com"}
		repo.On("GetByEmail", mock.Anything, "user@example.com").Return((*models.User)(nil), nil).Once()
		repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{
			ID:    "user-1",
			Name:  "User",
			Email: "user@example.com",
		}, nil).Once()
		handler := adminhandlers.NewCreateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users", internaltests.MarshalToJSON(t, request))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.CreateUserResponse](t, reqCtx)
		if payload.User == nil {
			t.Fatalf("expected user in payload, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestGetAllUsersHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		handler := adminhandlers.NewGetAllUsersHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users?limit=invalid", nil)

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid limit")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetAll", mock.Anything, (*string)(nil), 10).Return(([]models.User)(nil), (*string)(nil), constants.ErrForbidden).Once()
		handler := adminhandlers.NewGetAllUsersHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users", nil)

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		repo.AssertExpectations(t)
	})

	t.Run("success with cursor and limit", func(t *testing.T) {
		t.Parallel()

		cursor := "next-cursor"
		useCase, repo := admintests.NewUsersUseCaseFixture()
		queryCursor := "cur-1"
		repo.On("GetAll", mock.Anything, &queryCursor, 5).Return([]models.User{{ID: "user-1", Email: "u1@example.com"}}, &cursor, nil).Once()
		handler := adminhandlers.NewGetAllUsersHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users?cursor=cur-1&limit=5", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UsersPage](t, reqCtx)
		if payload.Users == nil {
			t.Fatalf("expected users field in payload, got %v", payload)
		}
		if len(payload.Users) != 1 {
			t.Fatalf("expected 1 user, got %d", len(payload.Users))
		}
		if payload.NextCursor == nil || *payload.NextCursor != cursor {
			t.Fatalf("expected next cursor to be %s, got %v", cursor, payload.NextCursor)
		}
		repo.AssertExpectations(t)
	})
}

func TestGetUserByIDHandler(t *testing.T) {
	t.Parallel()

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), constants.ErrUnauthorized).Once()
		handler := adminhandlers.NewGetUserByIDHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnauthorized, "unauthorized")
		repo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), nil).Once()
		handler := adminhandlers.NewGetUserByIDHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "user not found")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
		handler := adminhandlers.NewGetUserByIDHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetUserByIDResponse](t, reqCtx)
		if payload.User == nil {
			t.Fatalf("expected user in payload, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestUpdateUserHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		handler := adminhandlers.NewUpdateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		name := "Updated"
		request := types.UpdateUserRequest{Name: &name}
		repo.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), constants.ErrBadRequest).Once()
		handler := adminhandlers.NewUpdateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		name := "Updated"
		request := types.UpdateUserRequest{Name: &name}
		repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Name: "Old"}, nil).Once()
		repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(&models.User{ID: "user-1", Name: "Updated"}, nil).Once()
		handler := adminhandlers.NewUpdateUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpdateUserResponse](t, reqCtx)
		if payload.User == nil {
			t.Fatalf("expected user in payload, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestDeleteUserHandler(t *testing.T) {
	t.Parallel()

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
		repo.On("Delete", mock.Anything, "user-1").Return(constants.ErrBadRequest).Once()
		handler := adminhandlers.NewDeleteUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil).Once()
		repo.On("Delete", mock.Anything, "user-1").Return(nil).Once()
		handler := adminhandlers.NewDeleteUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.DeleteUserResponse](t, reqCtx)
		if payload.Message != "user deleted" {
			t.Fatalf("expected user deleted message, got %v", payload.Message)
		}
		repo.AssertExpectations(t)
	})
}
