package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestImpersonationUseCase_GetAllImpersonations(t *testing.T) {
	t.Parallel()

	useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
	now := time.Now().UTC()
	expected := []admintypes.Impersonation{{ID: "imp-1", ActorUserID: "actor-1", TargetUserID: "target-1", StartedAt: now, ExpiresAt: now.Add(time.Minute)}}
	impRepo.On("GetAllImpersonations", mock.Anything).Return(expected, nil).Once()

	list, err := useCase.GetAllImpersonations(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, list)
	impRepo.AssertExpectations(t)
}

func TestImpersonationUseCase_GetImpersonationByID(t *testing.T) {
	t.Parallel()

	t.Run("bad request when empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		_, err := useCase.GetImpersonationByID(context.Background(), "   ")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards trimmed id", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetImpersonationByID", mock.Anything, "imp-1").Return(&admintypes.Impersonation{ID: "imp-1"}, nil).Once()
		res, err := useCase.GetImpersonationByID(context.Background(), " imp-1 ")
		assert.NoError(t, err)
		assert.Equal(t, "imp-1", res.ID)
		impRepo.AssertExpectations(t)
	})

	t.Run("not found propagates", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		impRepo.On("GetImpersonationByID", mock.Anything, "imp-2").Return((*admintypes.Impersonation)(nil), constants.ErrNotFound).Once()
		_, err := useCase.GetImpersonationByID(context.Background(), "imp-2")
		assert.ErrorIs(t, err, constants.ErrNotFound)
		impRepo.AssertExpectations(t)
	})
}

func TestImpersonationUseCase_StartImpersonation(t *testing.T) {
	t.Parallel()

	t.Run("bad request when actor empty", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		_, err := useCase.StartImpersonation(context.Background(), "", nil, internaltests.PtrString("127.0.0.1"), internaltests.PtrString("user-agent"), admintypes.StartImpersonationRequest{TargetUserID: "t", Reason: "r"})
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("happy path returns result", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, sessionStateRepo, sessionSvc, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)

		ipAddress := internaltests.PtrString("127.0.0.1")
		userAgent := internaltests.PtrString("user-agent")
		impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
		impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
		tokenSvc.On("Generate").Return("tok", nil).Once()
		tokenSvc.On("Hash", "tok").Return("hash").Once()
		sessionSvc.On("Create", mock.Anything, "target-1", "hash", ipAddress, userAgent, mock.Anything).Return(&models.Session{ID: "sess"}, nil).Once()
		impRepo.On("CreateImpersonation", mock.Anything, mock.AnythingOfType("*types.Impersonation")).Return(nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*types.AdminSessionState")).Return(nil).Once()

		res, err := useCase.StartImpersonation(context.Background(), "actor-1", internaltests.PtrString("sess-actor"), ipAddress, userAgent, admintypes.StartImpersonationRequest{TargetUserID: "target-1", Reason: "reason"})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "sess", *res.SessionID)
		assert.Equal(t, "tok", *res.SessionToken)

		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
		sessionSvc.AssertExpectations(t)
		tokenSvc.AssertExpectations(t)
	})

	t.Run("error from token service propagates", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, _, _, tokenSvc := admintests.NewImpersonationUseCaseFixture(t)
		ipAddress := internaltests.PtrString("127.0.0.1")
		userAgent := internaltests.PtrString("user-agent")
		impRepo.On("UserExists", mock.Anything, "actor-1").Return(true, nil).Once()
		impRepo.On("UserExists", mock.Anything, "target-1").Return(true, nil).Once()
		tokenSvc.On("Generate").Return("", errors.New("fail")).Once()

		_, err := useCase.StartImpersonation(context.Background(), "actor-1", nil, ipAddress, userAgent, admintypes.StartImpersonationRequest{TargetUserID: "target-1", Reason: "reason"})
		assert.Error(t, err)
		impRepo.AssertExpectations(t)
		tokenSvc.AssertExpectations(t)
	})
}

func TestImpersonationUseCase_StopImpersonation(t *testing.T) {
	t.Parallel()

	t.Run("error when session state lookup fails", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return((*admintypes.AdminSessionState)(nil), constants.ErrNotFound).Once()
		err := useCase.StopImpersonation(context.Background(), "target-1", "impersonated-session-1", admintypes.StopImpersonationRequest{})
		assert.ErrorIs(t, err, constants.ErrNotFound)
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("unauthorized when session state has no impersonator", func(t *testing.T) {
		t.Parallel()

		useCase, _, sessionStateRepo, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1"}, nil).Once()

		err := useCase.StopImpersonation(context.Background(), "target-1", "impersonated-session-1", admintypes.StopImpersonationRequest{})
		assert.ErrorIs(t, err, constants.ErrUnauthorized)
		sessionStateRepo.AssertExpectations(t)
	})

	t.Run("not found when no active impersonation for resolved actor", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, sessionStateRepo, _, _ := admintests.NewImpersonationUseCaseFixture(t)
		actorID := "actor-1"
		sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1", ImpersonatorUserID: &actorID}, nil).Once()
		impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return((*admintypes.Impersonation)(nil), nil).Once()

		err := useCase.StopImpersonation(context.Background(), "target-1", "impersonated-session-1", admintypes.StopImpersonationRequest{})
		assert.ErrorIs(t, err, constants.ErrNotFound)
		sessionStateRepo.AssertExpectations(t)
		impRepo.AssertExpectations(t)
	})

	t.Run("successful stop uses impersonator from session state", func(t *testing.T) {
		t.Parallel()

		useCase, impRepo, sessionStateRepo, sessionSvc, _ := admintests.NewImpersonationUseCaseFixture(t)
		actorID := "actor-1"
		targetID := "target-1"

		imp := &admintypes.Impersonation{ID: "imp-1", ActorUserID: "actor-1", ImpersonationSessionID: internaltests.PtrString("sess")}
		sessionStateRepo.On("GetBySessionID", mock.Anything, "impersonated-session-1").Return(&admintypes.AdminSessionState{SessionID: "impersonated-session-1", ImpersonatorUserID: &actorID}, nil).Once()
		impRepo.On("GetLatestActiveImpersonationByActor", mock.Anything, "actor-1").Return(imp, nil).Once()
		sessionStateRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
		sessionSvc.On("Delete", mock.Anything, "sess").Return(nil).Once()
		impRepo.On("EndImpersonation", mock.Anything, "imp-1", mock.AnythingOfType("*string")).Return(nil).Once()

		err := useCase.StopImpersonation(context.Background(), targetID, "impersonated-session-1", admintypes.StopImpersonationRequest{})
		assert.NoError(t, err)

		impRepo.AssertExpectations(t)
		sessionStateRepo.AssertExpectations(t)
		sessionSvc.AssertExpectations(t)
	})
}
