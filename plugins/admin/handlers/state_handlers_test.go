package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetUserStateHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetUserState", mock.Anything, "user-1").Return(nil, errors.New("forbidden")).Once()
		handler := NewGetUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("not found on nil state", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetUserState", mock.Anything, "user-1").Return(nil, nil).Once()
		handler := NewGetUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "user state not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetUserState", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", IsBanned: false}, nil).Once()
		handler := NewGetUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestUpsertUserStateHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		handler := NewUpsertUserStateHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.UpsertUserStateRequest{IsBanned: true}
		actorID := "actor-1"
		useCase.On("UpsertUserState", mock.Anything, "user-1", request, &actorID).Return(nil, errors.New("invalid state")).Once()
		handler := NewUpsertUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid state")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.UpsertUserStateRequest{IsBanned: true}
		useCase.On("UpsertUserState", mock.Anything, "user-1", request, (*string)(nil)).Return(&types.AdminUserState{UserID: "user-1", IsBanned: true}, nil).Once()
		handler := NewUpsertUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestDeleteUserStateHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("DeleteUserState", mock.Anything, "user-1").Return(errors.New("not found")).Once()
		handler := NewDeleteUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("DeleteUserState", mock.Anything, "user-1").Return(nil).Once()
		handler := NewDeleteUserStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "user state deleted" {
			t.Fatalf("expected user state deleted message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetBannedUserStatesHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetBannedUserStates", mock.Anything).Return(nil, errors.New("internal error")).Once()
		handler := NewGetBannedUserStatesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/banned", nil)
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetBannedUserStates", mock.Anything).Return([]types.AdminUserState{{UserID: "user-1", IsBanned: true}}, nil).Once()
		handler := NewGetBannedUserStatesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/banned", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestBanUserHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		handler := NewBanUserHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.BanUserRequest{}
		actorID := "actor-1"
		useCase.On("BanUser", mock.Anything, "user-1", request, &actorID).Return(nil, errors.New("cannot ban"))
		handler := NewBanUserHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "cannot ban")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.BanUserRequest{}
		useCase.On("BanUser", mock.Anything, "user-1", request, (*string)(nil)).Return(&types.AdminUserState{UserID: "user-1", IsBanned: true}, nil)
		handler := NewBanUserHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestUnbanUserHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("UnbanUser", mock.Anything, "user-1").Return(nil, errors.New("not found")).Once()
		handler := NewUnbanUserHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/unban", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("UnbanUser", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", IsBanned: false}, nil).Once()
		handler := NewUnbanUserHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/unban", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetSessionStateHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetSessionState", mock.Anything, "session-1").Return(nil, errors.New("forbidden")).Once()
		handler := NewGetSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("not found on nil", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetSessionState", mock.Anything, "session-1").Return(nil, nil).Once()
		handler := NewGetSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "session state not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetSessionState", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
		handler := NewGetSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestUpsertSessionStateHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		handler := NewUpsertSessionStateHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", []byte("{invalid"))
		req.SetPathValue("session_id", "session-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.UpsertSessionStateRequest{Revoke: true}
		actorID := "actor-1"
		useCase.On("UpsertSessionState", mock.Anything, "session-1", request, &actorID).Return(nil, errors.New("invalid request")).Once()
		handler := NewUpsertSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", mustJSON(t, request))
		req.SetPathValue("session_id", "session-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid request")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		request := types.UpsertSessionStateRequest{Revoke: true}
		useCase.On("UpsertSessionState", mock.Anything, "session-1", request, (*string)(nil)).Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
		handler := NewUpsertSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", mustJSON(t, request))
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestDeleteSessionStateHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("DeleteSessionState", mock.Anything, "session-1").Return(errors.New("not found")).Once()
		handler := NewDeleteSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("DeleteSessionState", mock.Anything, "session-1").Return(nil).Once()
		handler := NewDeleteSessionStateHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "session state deleted" {
			t.Fatalf("expected session state deleted message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetRevokedSessionStatesHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetRevokedSessionStates", mock.Anything).Return(nil, errors.New("internal error")).Once()
		handler := NewGetRevokedSessionStatesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/sessions/revoked", nil)
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetRevokedSessionStates", mock.Anything).Return([]types.AdminSessionState{{SessionID: "session-1"}}, nil).Once()
		handler := NewGetRevokedSessionStatesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/sessions/revoked", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetUserAdminSessionsHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		useCase.On("GetUserAdminSessions", mock.Anything, "user-1").Return(nil, errors.New("not found")).Once()
		handler := NewGetUserAdminSessionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1/sessions", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		expiresAt := time.Now().UTC().Add(time.Hour)
		useCase.On("GetUserAdminSessions", mock.Anything, "user-1").Return([]types.AdminUserSession{{Session: models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: expiresAt}}}, nil).Once()
		handler := NewGetUserAdminSessionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1/sessions", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestRevokeSessionHandler(t *testing.T) {
	t.Run("invalid json defaults payload", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		actorID := "actor-1"
		useCase.On("RevokeSession", mock.Anything, "session-1", (*string)(nil), &actorID).Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
		handler := NewRevokeSessionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", []byte("{invalid"))
		req.SetPathValue("session_id", "session-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		useCase.AssertExpectations(t)
	})

	t.Run("use case error", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		reason := "security"
		useCase.On("RevokeSession", mock.Anything, "session-1", &reason, (*string)(nil)).Return(nil, errors.New("forbidden")).Once()
		handler := NewRevokeSessionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", mustJSON(t, types.RevokeSessionRequest{Reason: &reason}))
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockStateUseCase{}
		reason := "suspicious"
		useCase.On("RevokeSession", mock.Anything, "session-1", &reason, (*string)(nil)).Return(&types.AdminSessionState{SessionID: "session-1", RevokedReason: &reason}, nil).Once()
		handler := NewRevokeSessionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", mustJSON(t, types.RevokeSessionRequest{Reason: &reason}))
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}
