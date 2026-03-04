package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetAllUsersHandler(t *testing.T) {
	t.Run("invalid limit", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		handler := NewGetAllUsersHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users?limit=invalid", nil)

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid limit")
		useCase.AssertExpectations(t)
	})

	t.Run("use case error", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("GetAll", mock.Anything, (*string)(nil), 20).Return(nil, errors.New("forbidden")).Once()
		handler := NewGetAllUsersHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users", nil)

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("success with cursor and limit", func(t *testing.T) {
		cursor := "next-cursor"
		queryCursor := "cur-1"
		useCase := &mockUsersUseCase{}
		useCase.On("GetAll", mock.Anything, &queryCursor, 5).Return(&types.UsersPage{Users: []models.User{{ID: "user-1", Email: "u1@example.com"}}, NextCursor: &cursor}, nil).Once()
		handler := NewGetAllUsersHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users?cursor=cur-1&limit=5", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["users"]; !ok {
			t.Fatalf("expected users key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetUserByIDHandler(t *testing.T) {
	t.Run("use case error", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("GetByID", mock.Anything, "user-1").Return(nil, errors.New("unauthorized")).Once()
		handler := NewGetUserByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnauthorized, "unauthorized")
		useCase.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
		handler := NewGetUserByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "user not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
		handler := NewGetUserByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["user"]; !ok {
			t.Fatalf("expected user key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestCreateUserHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		handler := NewCreateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users", []byte("{invalid"))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
		useCase.AssertExpectations(t)
	})

	t.Run("use case error", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		request := types.CreateUserRequest{Name: "User", Email: "user@example.com"}
		useCase.On("Create", mock.Anything, request).Return(nil, errors.New("email already exists")).Once()
		handler := NewCreateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users", mustJSON(t, request))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "email already exists")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		request := types.CreateUserRequest{Name: "User", Email: "user@example.com"}
		useCase.On("Create", mock.Anything, request).Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
		handler := NewCreateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users", mustJSON(t, request))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["user"]; !ok {
			t.Fatalf("expected user key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestUpdateUserHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		handler := NewUpdateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
		useCase.AssertExpectations(t)
	})

	t.Run("use case error", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		name := "Updated"
		request := types.UpdateUserRequest{Name: &name}
		useCase.On("Update", mock.Anything, "user-1", request).Return(nil, errors.New("not found")).Once()
		handler := NewUpdateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		name := "Updated"
		request := types.UpdateUserRequest{Name: &name}
		useCase.On("Update", mock.Anything, "user-1", request).Return(&models.User{ID: "user-1", Name: name}, nil).Once()
		handler := NewUpdateUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/users/user-1", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["user"]; !ok {
			t.Fatalf("expected user key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestDeleteUserHandler(t *testing.T) {
	t.Run("use case error", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("Delete", mock.Anything, "user-1").Return(errors.New("cannot delete"))
		handler := NewDeleteUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "cannot delete")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUsersUseCase{}
		useCase.On("Delete", mock.Anything, "user-1").Return(nil)
		handler := NewDeleteUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "user deleted" {
			t.Fatalf("expected user deleted message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}
