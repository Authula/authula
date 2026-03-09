package handlers_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetUserStateHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), constants.ErrBadRequest).Once()
		handler := adminhandlers.NewGetUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		userStateRepo.AssertExpectations(t)
	})

	t.Run("not found on nil state", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), nil).Once()
		handler := adminhandlers.NewGetUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "user state not found")
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", IsBanned: false}, nil).Once()
		handler := adminhandlers.NewGetUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetUserStateResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		userStateRepo.AssertExpectations(t)
	})
}

func TestUpsertUserStateHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		handler := adminhandlers.NewUpsertUserStateHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		request := types.UpsertUserStateRequest{IsBanned: true}
		actorID := "actor-1"
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(constants.ErrBadRequest).Once()
		handler := adminhandlers.NewUpsertUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		request := types.UpsertUserStateRequest{IsBanned: true}
		result := &types.AdminUserState{UserID: "user-1", IsBanned: true}
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(result, nil).Once()
		handler := adminhandlers.NewUpsertUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/users/user-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpsertUserStateResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})
}

func TestDeleteUserStateHandler(t *testing.T) {
	t.Parallel()
	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("Delete", mock.Anything, "user-1").Return(constants.ErrNotFound).Once()
		handler := adminhandlers.NewDeleteUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("Delete", mock.Anything, "user-1").Return(nil).Once()
		handler := adminhandlers.NewDeleteUserStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/users/user-1", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.DeleteUserStateResponse](t, reqCtx)
		if payload.Message != "user state deleted" {
			t.Fatalf("expected user state deleted message, got %v", payload.Message)
		}
		userStateRepo.AssertExpectations(t)
	})
}

func TestGetBannedUserStatesHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetBanned", mock.Anything).Return(([]types.AdminUserState)(nil), errors.New("internal error")).Once()
		handler := adminhandlers.NewGetBannedUserStatesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/banned", nil)
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, _ := admintests.NewStateUseCaseFixture()
		userStateRepo.On("GetBanned", mock.Anything).Return([]types.AdminUserState{{UserID: "user-1", IsBanned: true}}, nil).Once()
		handler := adminhandlers.NewGetBannedUserStatesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/banned", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.AdminUserState](t, reqCtx)
		if payload == nil || len(payload) != 1 {
			t.Fatalf("expected banned user state, got %v", payload)
		}
		userStateRepo.AssertExpectations(t)
	})
}

func TestBanUserHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		handler := adminhandlers.NewBanUserHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		request := types.BanUserRequest{}
		actorID := "actor-1"
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(constants.ErrBadRequest).Once()
		handler := adminhandlers.NewBanUserHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		request := types.BanUserRequest{}
		result := &types.AdminUserState{UserID: "user-1", IsBanned: true}
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(result, nil).Once()
		handler := adminhandlers.NewBanUserHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/ban", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpsertUserStateResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})
}

func TestUnbanUserHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(constants.ErrNotFound).Once()
		handler := adminhandlers.NewUnbanUserHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/unban", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, userStateRepo, _, impRepo := admintests.NewStateUseCaseFixture()
		result := &types.AdminUserState{UserID: "user-1", IsBanned: false}
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		userStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminUserState")).Return(nil).Once()
		userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(result, nil).Once()
		handler := adminhandlers.NewUnbanUserHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/users/user-1/unban", nil)
		req.SetPathValue("user_id", "user-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UnbanUserResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		impRepo.AssertExpectations(t)
		userStateRepo.AssertExpectations(t)
	})
}

func TestGetSessionStateHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), constants.ErrForbidden).Once()
		handler := adminhandlers.NewGetSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("not found on nil", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return((*types.AdminSessionState)(nil), nil).Once()
		handler := adminhandlers.NewGetSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "session state not found")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
		handler := adminhandlers.NewGetSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetSessionStateResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestUpsertSessionStateHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		handler := adminhandlers.NewUpsertSessionStateHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", []byte("{invalid"))
		req.SetPathValue("session_id", "session-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		request := types.UpsertSessionStateRequest{Revoke: true}
		actorID := "actor-1"
		sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(constants.ErrBadRequest).Once()
		handler := adminhandlers.NewUpsertSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("session_id", "session-1")
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		request := types.UpsertSessionStateRequest{Revoke: true}
		sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1"}, nil).Once()
		handler := adminhandlers.NewUpsertSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/admin/states/sessions/session-1", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("session_id", "session-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpsertSessionStateResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected state, got %v", payload)
		}
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestDeleteSessionStateHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("Delete", mock.Anything, "session-1").Return(constants.ErrNotFound).Once()
		handler := adminhandlers.NewDeleteSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("Delete", mock.Anything, "session-1").Return(nil).Once()
		handler := adminhandlers.NewDeleteSessionStateHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/states/sessions/session-1", nil)
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.DeleteSessionStateResponse](t, reqCtx)
		if payload.Message != "session state deleted" {
			t.Fatalf("expected session state deleted message, got %v", payload.Message)
		}
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestGetRevokedSessionStatesHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetRevoked", mock.Anything).Return(([]types.AdminSessionState)(nil), errors.New("internal error")).Once()
		handler := adminhandlers.NewGetRevokedSessionStatesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/revoked", nil)
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		sessionStateRepo.On("GetRevoked", mock.Anything).Return([]types.AdminSessionState{{SessionID: "session-1"}}, nil).Once()
		handler := adminhandlers.NewGetRevokedSessionStatesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/sessions/revoked", nil)
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.AdminSessionState](t, reqCtx)
		if payload == nil || len(payload) != 1 {
			t.Fatalf("expected session state, got %v", payload)
		}
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestGetUserAdminSessionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		sessionStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(([]types.AdminUserSession)(nil), constants.ErrNotFound).Once()
		handler := adminhandlers.NewGetUserAdminSessionsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1/sessions", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, impRepo := admintests.NewStateUseCaseFixture()
		expiresAt := time.Now().UTC().Add(time.Hour)
		impRepo.On("UserExists", mock.Anything, "user-1").Return(true, nil).Once()
		sessionStateRepo.On("GetByUserID", mock.Anything, "user-1").Return([]types.AdminUserSession{{Session: models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: expiresAt}}}, nil).Once()
		handler := adminhandlers.NewGetUserAdminSessionsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/states/users/user-1/sessions", nil)
		req.SetPathValue("user_id", "user-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.AdminUserSession](t, reqCtx)
		if payload == nil || len(payload) != 1 {
			t.Fatalf("expected user sessions, got %v", payload)
		}
		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
	})
}

func TestRevokeSessionHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _ := admintests.NewStateUseCaseFixture()
		handler := adminhandlers.NewRevokeSessionHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", []byte("{invalid"))
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		reason := "security"
		sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(constants.ErrForbidden).Once()
		handler := adminhandlers.NewRevokeSessionHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", internaltests.MarshalToJSON(t, types.RevokeSessionRequest{Reason: &reason}))
		req.SetPathValue("session_id", "session-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _ := admintests.NewStateUseCaseFixture()
		reason := "suspicious"
		sessionStateRepo.On("SessionExists", mock.Anything, "session-1").Return(true, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()
		sessionStateRepo.On("GetBySessionID", mock.Anything, "session-1").Return(&types.AdminSessionState{SessionID: "session-1", RevokedReason: &reason}, nil).Once()
		handler := adminhandlers.NewRevokeSessionHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/states/sessions/session-1/revoke", internaltests.MarshalToJSON(t, types.RevokeSessionRequest{Reason: &reason}))
		req.SetPathValue("session_id", "session-1")
		actorID := "actor-1"
		reqCtx.UserID = &actorID
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.RevokeSessionResponse](t, reqCtx)
		if payload.State == nil {
			t.Fatalf("expected session state, got %v", payload)
		}
		sessionStateRepo.AssertExpectations(t)
	})
}
