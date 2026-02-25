package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type usersUseCaseStub struct {
	getAllUsersFn func(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error)
	createUserFn  func(ctx context.Context, request types.CreateUserRequest) (*models.User, error)
	getUserByIDFn func(ctx context.Context, userID string) (*models.User, error)
	updateUserFn  func(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error)
	deleteUserFn  func(ctx context.Context, userID string) error
}

func (s usersUseCaseStub) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	return s.getAllUsersFn(ctx, cursor, limit)
}

func (s usersUseCaseStub) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	return s.createUserFn(ctx, request)
}

func (s usersUseCaseStub) GetByID(ctx context.Context, userID string) (*models.User, error) {
	return s.getUserByIDFn(ctx, userID)
}

func (s usersUseCaseStub) Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	return s.updateUserFn(ctx, userID, request)
}

func (s usersUseCaseStub) Delete(ctx context.Context, userID string) error {
	return s.deleteUserFn(ctx, userID)
}

func withReqCtx(req *http.Request) (*http.Request, *models.RequestContext) {
	rc := &models.RequestContext{Request: req, Values: map[string]any{}, Route: &models.Route{Method: req.Method, Path: req.URL.Path}}
	ctx := models.NewContextWithRequestContext(req.Context(), rc)
	req = req.WithContext(ctx)
	return req, rc
}

func TestGetAllUsersHandler_SuccessWithCursor(t *testing.T) {
	stub := usersUseCaseStub{
		getAllUsersFn: func(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
			if limit != 2 {
				t.Fatalf("expected limit 2, got %d", limit)
			}
			if cursor == nil || *cursor != "u-002" {
				t.Fatalf("expected cursor u-002")
			}
			users := []models.User{
				{ID: "u-001"},
				{ID: "u-002"},
			}
			next := "u-003"
			return &types.UsersPage{Users: users, NextCursor: &next}, nil
		},
		createUserFn:  func(ctx context.Context, request types.CreateUserRequest) (*models.User, error) { return nil, nil },
		getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
			return nil, nil
		},
		deleteUserFn: func(ctx context.Context, userID string) error { return nil },
	}

	h := NewGetAllUsersHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/admin/users?cursor=u-002&limit=2", nil)
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rc.ResponseStatus)
	}

	body := string(rc.ResponseBody)
	if !strings.Contains(body, `"id":"u-001"`) || !strings.Contains(body, `"id":"u-002"`) {
		t.Fatalf("response body missing expected user IDs: %s", body)
	}
	if !strings.Contains(body, `"next_cursor":"u-003"`) {
		t.Fatalf("response body missing expected next_cursor: %s", body)
	}
}

func TestCreateUserHandler_Success(t *testing.T) {
	stub := usersUseCaseStub{
		getAllUsersFn: func(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) { return nil, nil },
		createUserFn: func(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
			if request.Name != "John Doe" || request.Email != "john.doe@example.com" {
				t.Fatalf("unexpected create request: %+v", request)
			}
			return &models.User{ID: "u-22", Name: request.Name, Email: request.Email}, nil
		},
		getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
			return nil, nil
		},
		deleteUserFn: func(ctx context.Context, userID string) error { return nil },
	}

	h := NewCreateUserHandler(stub)
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(`{"name":"John Doe","email":"john.doe@example.com"}`))
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rc.ResponseStatus)
	}

	body := string(rc.ResponseBody)
	expectedKVs := []string{
		`"id":"u-22"`,
		`"name":"John Doe"`,
		`"email":"john.doe@example.com"`,
	}
	for _, kv := range expectedKVs {
		if !strings.Contains(body, kv) {
			t.Fatalf("response body missing %s", kv)
		}
	}
}

func TestUpdateUserHandler_BadJSON(t *testing.T) {
	stub := usersUseCaseStub{
		getAllUsersFn: func(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) { return nil, nil },
		createUserFn:  func(ctx context.Context, request types.CreateUserRequest) (*models.User, error) { return nil, nil },
		getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
			return nil, nil
		},
		deleteUserFn: func(ctx context.Context, userID string) error { return nil },
	}

	h := NewUpdateUserHandler(stub)
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/u-1", strings.NewReader("{"))
	req.SetPathValue("user_id", "u-1")
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rc.ResponseStatus)
	}

	body := string(rc.ResponseBody)
	if !strings.Contains(body, "invalid request body") {
		t.Fatalf("response body missing expected error message: %s", body)
	}
}

func TestDeleteUserHandler_NotFound(t *testing.T) {
	stub := usersUseCaseStub{
		getAllUsersFn: func(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) { return nil, nil },
		createUserFn:  func(ctx context.Context, request types.CreateUserRequest) (*models.User, error) { return nil, nil },
		getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
			return nil, nil
		},
		deleteUserFn: func(ctx context.Context, userID string) error { return context.Canceled },
	}

	h := NewDeleteUserHandler(stub)
	req := httptest.NewRequest(http.MethodDelete, "/admin/users/u-missing", nil)
	req.SetPathValue("user_id", "u-missing")
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rc.ResponseStatus)
	}

	body := string(rc.ResponseBody)
	if body == "" {
		t.Fatalf("response body is empty, expected error message")
	}
}
