package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func newImpersonationServiceFixture() (*adminservices.ImpersonationService, *admintests.MockImpersonationRepository, *admintests.MockSessionStateRepository, *internaltests.MockSessionService, *internaltests.MockTokenService) {
	svc, impRepo, sessRepo, sessSvc, tokSvc := admintests.NewImpersonationServiceFixture()
	return svc, impRepo, sessRepo, sessSvc, tokSvc
}

func TestImpersonationService_StartImpersonation_validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name  string
		actor string
		req   admintypes.StartImpersonationRequest
		setup func(impRepo *admintests.MockImpersonationRepository)
		want  error
	}{
		{name: "empty actor", actor: "", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, want: internalerrors.ErrBadRequest},
		{name: "empty target", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "  ", Reason: "r"}, want: internalerrors.ErrBadRequest},
		{name: "same user", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "a1", Reason: "r"}, want: internalerrors.ErrBadRequest},
		{name: "empty reason", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "   "}, want: internalerrors.ErrBadRequest},
		{name: "actor not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(false, nil).Once()
		}, want: internalerrors.ErrNotFound},
		{name: "target not exists", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r"}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(false, nil).Once()
		}, want: internalerrors.ErrNotFound},
		{name: "expires invalid zero", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: func(i int) *int { return &i }(0)}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, want: internalerrors.ErrBadRequest},
		{name: "expires invalid large", actor: "a1", req: admintypes.StartImpersonationRequest{TargetUserID: "u2", Reason: "r", ExpiresInSeconds: func(i int) *int { return &i }(999999)}, setup: func(impRepo *admintests.MockImpersonationRepository) {
			impRepo.On("UserExists", mock.Anything, "a1").Return(true, nil).Once()
			impRepo.On("UserExists", mock.Anything, "u2").Return(true, nil).Once()
		}, want: internalerrors.ErrBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, impRepo, _, _, _ := newImpersonationServiceFixture()

			if tc.setup != nil {
				tc.setup(impRepo)
			}

			ipAddress := new("127.0.0.1")
			userAgent := new("user-agent")
			_, err := svc.StartImpersonation(ctx, internaltests.TestActor(), tc.actor, nil, ipAddress, userAgent, tc.req, nil, "", 0)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			} else {
				require.NoError(t, err)
			}
			impRepo.AssertExpectations(t)
		})
	}
}

func TestImpersonationService_StartImpersonation_success(t *testing.T) {
	t.Parallel()

	svc, impRepo, sessRepo, sessSvc, tokSvc := newImpersonationServiceFixture()
	ctx := context.Background()

	// happy path with session creation
	impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
	impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()

	rawToken := "rawtoken"
	tokSvc.On("Generate").Return(rawToken, nil).Once()
	tokSvc.On("Hash", rawToken).Return("hashed").Once()
	sessSvc.On("Create", mock.Anything, "target", "hashed", mock.Anything, mock.Anything, mock.Anything).Return(&models.Session{ID: "sess1"}, nil).Once()

	// after create impersonation and state upsert
	impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()
	sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()

	req := admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason", ExpiresInSeconds: func(i int) *int { return &i }(60)}
	ipAddress := new("127.0.0.1")
	userAgent := new("user-agent")
	res, err := svc.StartImpersonation(ctx, internaltests.TestActor(), "actor", nil, ipAddress, userAgent, req, []string{"scope1"}, "orig-token", 100)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.SessionID)
	require.Equal(t, "sess1", *res.SessionID)
	require.NotNil(t, res.SessionToken)
	require.Equal(t, rawToken, *res.SessionToken)
	require.Equal(t, "actor", res.ImpersonatorUserID)
	require.Equal(t, []string{"scope1"}, res.ImpersonatorScopes)
	require.Equal(t, "target", res.TargetUserID)
	require.Equal(t, "orig-token", res.OriginalCookieToken)

	impRepo.AssertExpectations(t)
	sessRepo.AssertExpectations(t)
	sessSvc.AssertExpectations(t)
	tokSvc.AssertExpectations(t)
}

func TestImpersonationService_StartImpersonation_noSessionServices(t *testing.T) {
	t.Parallel()

	impRepo := &admintests.MockImpersonationRepository{}
	sessRepo := &admintests.MockSessionStateRepository{}
	// Use constructor to create service with nil session/token services
	svc := adminservices.NewImpersonationService(impRepo, sessRepo, nil, nil, time.Minute, time.Minute)
	ctx := context.Background()

	impRepo.On("UserExists", mock.Anything, "actor").Return(true, nil).Once()
	impRepo.On("UserExists", mock.Anything, "target").Return(true, nil).Once()
	impRepo.On("CreateImpersonation", mock.Anything, mock.Anything).Return(nil).Once()

	req := admintypes.StartImpersonationRequest{TargetUserID: "target", Reason: "reason"}
	ipAddress := new("127.0.0.1")
	userAgent := new("user-agent")
	res, err := svc.StartImpersonation(ctx, internaltests.TestActor(), "actor", nil, ipAddress, userAgent, req, nil, "", 0)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Nil(t, res.SessionID)
	require.Nil(t, res.SessionToken)

	impRepo.AssertExpectations(t)
	sessRepo.AssertExpectations(t)
}

func TestImpersonationService_ValidateImpersonationCookie(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("session not found returns forbidden", func(t *testing.T) {
		t.Parallel()
		svc, _, _, sessSvc, tokSvc := newImpersonationServiceFixture()
		tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
		sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return((*models.Session)(nil), nil).Once()
		_, err := svc.ValidateImpersonationCookie(ctx, "orig-token")
		require.ErrorIs(t, err, internalerrors.ErrForbidden)
		tokSvc.AssertExpectations(t)
		sessSvc.AssertExpectations(t)
	})

	t.Run("success returns session", func(t *testing.T) {
		t.Parallel()
		svc, _, _, sessSvc, tokSvc := newImpersonationServiceFixture()
		tokSvc.On("Hash", "orig-token").Return("hashed-orig").Once()
		sessSvc.On("GetByToken", mock.Anything, "hashed-orig").Return(&models.Session{ID: "orig-sess", UserID: "actor-1"}, nil).Once()
		sess, err := svc.ValidateImpersonationCookie(ctx, "orig-token")
		require.NoError(t, err)
		require.NotNil(t, sess)
		require.Equal(t, "orig-sess", sess.ID)
		tokSvc.AssertExpectations(t)
		sessSvc.AssertExpectations(t)
	})
}

func TestImpersonationService_StopImpersonation(t *testing.T) {
	t.Parallel()

	svc, impRepo, sessRepo, sessSvc, _ := newImpersonationServiceFixture()
	ctx := context.Background()

	imp := &admintypes.Impersonation{ID: "imp1", ActorUserID: "actor", TargetUserID: "target", ImpersonationSessionID: admintests.PtrString(t, "sess1"), Reason: "r", ExpiresAt: time.Now().UTC()}

	// case: empty actor
	err := svc.StopImpersonation(ctx, internaltests.TestActor(), "", admintypes.StopImpersonationRequest{})
	require.ErrorIs(t, err, internalerrors.ErrBadRequest)

	// case: id not found
	impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp1").Return(nil, nil).Once()
	err = svc.StopImpersonation(ctx, internaltests.TestActor(), "actor", admintypes.StopImpersonationRequest{ImpersonationID: admintests.PtrString(t, "imp1")})
	require.ErrorIs(t, err, internalerrors.ErrNotFound)
	impRepo.AssertExpectations(t)

	// case: found but wrong actor
	impRepo.ExpectedCalls = nil
	impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp1").Return(imp, nil).Once()
	err = svc.StopImpersonation(ctx, internaltests.TestActor(), "other", admintypes.StopImpersonationRequest{ImpersonationID: admintests.PtrString(t, "imp1")})
	require.ErrorIs(t, err, internalerrors.ErrForbidden)
	impRepo.AssertExpectations(t)

	// case: success with session cleanup
	impRepo.ExpectedCalls = nil
	sessRepo.ExpectedCalls = nil
	sessSvc.ExpectedCalls = nil

	impRepo.On("GetActiveImpersonationByID", mock.Anything, "imp1").Return(imp, nil).Once()
	sessRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil).Once()
	sessSvc.On("Delete", mock.Anything, "sess1").Return(nil).Once()
	impRepo.On("EndImpersonation", mock.Anything, "imp1", mock.Anything).Return(nil).Once()

	err = svc.StopImpersonation(ctx, internaltests.TestActor(), "actor", admintypes.StopImpersonationRequest{ImpersonationID: admintests.PtrString(t, "imp1")})
	require.NoError(t, err)
	impRepo.AssertExpectations(t)
	sessRepo.AssertExpectations(t)
	sessSvc.AssertExpectations(t)
}

func TestImpersonationService_GetAllImpersonations(t *testing.T) {
	t.Parallel()

	svc, impRepo, _, _, _ := newImpersonationServiceFixture()
	ctx := context.Background()

	list := []admintypes.Impersonation{{ID: "i1"}}
	impRepo.On("GetAllImpersonations", mock.Anything).Return(list, nil).Once()
	res, err := svc.GetAllImpersonations(ctx, internaltests.TestActor())
	require.NoError(t, err)
	require.Len(t, res, 1)
	impRepo.AssertExpectations(t)
}

func TestImpersonationService_GetImpersonationByID(t *testing.T) {
	t.Parallel()

	svc, impRepo, _, _, _ := newImpersonationServiceFixture()
	ctx := context.Background()

	_, err := svc.GetImpersonationByID(ctx, internaltests.TestActor(), "   ")
	require.ErrorIs(t, err, internalerrors.ErrBadRequest)

	impRepo.On("GetImpersonationByID", mock.Anything, "i1").Return(nil, nil).Once()
	_, err = svc.GetImpersonationByID(ctx, internaltests.TestActor(), "i1")
	require.ErrorIs(t, err, internalerrors.ErrNotFound)

	impObj := &admintypes.Impersonation{ID: "i1"}
	impRepo.ExpectedCalls = nil
	impRepo.On("GetImpersonationByID", mock.Anything, "i1").Return(impObj, nil).Once()
	res, err := svc.GetImpersonationByID(ctx, internaltests.TestActor(), " i1 ")
	require.NoError(t, err)
	require.Equal(t, "i1", res.ID)
	impRepo.AssertExpectations(t)
}
