package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type invitationRoleErrorAccessControl struct {
	*orgtests.AccessControlServiceStub
	err error
}

func (s *invitationRoleErrorAccessControl) RoleExists(context.Context, string) (bool, error) {
	return false, s.err
}

// invitationSkippingTxRunner returns success without running the transaction
// callback, which lets the caller observe an empty transaction result.
type invitationSkippingTxRunner struct{}

func (invitationSkippingTxRunner) RunInTx(context.Context, *sql.TxOptions, func(context.Context, bun.Tx) error) error {
	return nil
}

type invitationSignalingLogger struct {
	mu     sync.Mutex
	errors []string
	done   chan struct{}
	once   sync.Once
}

func newInvitationSignalingLogger() *invitationSignalingLogger {
	return &invitationSignalingLogger{done: make(chan struct{})}
}

func (l *invitationSignalingLogger) Debug(string, ...any) {}
func (l *invitationSignalingLogger) Info(string, ...any)  {}
func (l *invitationSignalingLogger) Warn(string, ...any)  {}

func (l *invitationSignalingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.errors = append(l.errors, msg)
	l.mu.Unlock()
	l.once.Do(func() { close(l.done) })
}

func (l *invitationSignalingLogger) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errors...)
}

func waitForInvitationLog(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func newInvitationHooksExecutor(configure func(*orgtypes.OrganizationInvitationServiceHooks)) *ServiceHookExecutor {
	hooks := &orgtypes.OrganizationInvitationServiceHooks{}
	if configure != nil {
		configure(hooks)
	}
	return NewServiceHookExecutor(&orgtypes.OrganizationsServiceHooksConfig{Invitations: hooks}, nil, nil)
}

func newInvitationServiceForTest(f *invitationExtraFixture, txRunner organizationInvitationTxRunner, executor *ServiceHookExecutor) *organizationInvitationService {
	return newInvitationServiceWithOverrides(f, txRunner, executor, f.logger, f.accessControl)
}

func newInvitationServiceWithOverrides(f *invitationExtraFixture, txRunner organizationInvitationTxRunner, executor *ServiceHookExecutor, logger models.Logger, accessControl rootservices.AccessControlService) *organizationInvitationService {
	return NewOrganizationInvitationService(
		txRunner,
		&models.Config{BaseURL: "https://example.com", BasePath: "/auth"},
		f.pluginConfig,
		logger,
		nil,
		f.userSvc,
		f.mailer,
		accessControl,
		f.orgRepo,
		f.invRepo,
		f.memberRepo,
		&ServiceUtils{orgRepo: f.orgRepo, orgMemberRepo: f.memberRepo},
		f.templateMgr,
		executor,
	)
}

func expectInvitationOrganizationAccess(f *invitationExtraFixture, organizationID, userID string) {
	f.orgRepo.On("GetByID", mock.Anything, organizationID).Return(&orgtypes.Organization{ID: organizationID, OwnerID: "owner-1", Name: "Acme"}, nil).Once()
	f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, organizationID, userID).
		Return(&orgtypes.OrganizationMember{ID: "mem-actor", OrganizationID: organizationID, UserID: userID, Role: "admin"}, nil).
		Once()
}

func pendingInvitation(id string) *orgtypes.OrganizationInvitation {
	return &orgtypes.OrganizationInvitation{
		ID:             id,
		OrganizationID: "org-1",
		Email:          "user@example.com",
		Role:           "member",
		Status:         orgtypes.OrganizationInvitationStatusPending,
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
}

func TestOrganizationInvitationService_CreateExtraBranches(t *testing.T) {
	t.Parallel()

	errLookup := errors.New("existing invitation lookup failed")
	errMail := errors.New("mailer unavailable")

	t.Run("existing invitation lookup error is returned", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", orgtypes.OrganizationInvitationStatusPending).Return(nil, errLookup).Once()

		invitation, err := f.svc.CreateOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}, "")

		require.ErrorIs(t, err, errLookup)
		require.Nil(t, invitation)
	})

	t.Run("background email error is logged", func(t *testing.T) {
		t.Parallel()

		logger := newInvitationSignalingLogger()
		f := newInvitationExtraFixture(t, nil, func(f *invitationExtraFixture) {
			f.mailer = &capturingMailer{err: errMail}
		})
		svc := newInvitationServiceWithOverrides(f, &orgtests.MockOrganizationInvitationTxRunner{}, nil, logger, f.accessControl)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", orgtypes.OrganizationInvitationStatusPending).Return(nil, nil).Once()
		f.invRepo.On("Create", mock.Anything, mock.Anything).Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := svc.CreateOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}, "")

		require.NoError(t, err)
		require.NotNil(t, invitation)
		waitForInvitationLog(t, logger.done, "background email error")
		require.Contains(t, strings.Join(logger.messages(), "\n"), "failed to send organization invitation email via built-in email service")
	})
}

func TestOrganizationInvitationService_GetAndRevokeExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")

	t.Run("get rejects an empty invitation id", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		invitation, err := f.svc.GetOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "")

		require.ErrorIs(t, err, coreerrors.ErrUnauthorized)
		require.Nil(t, invitation)
	})

	t.Run("get rejects actors without organization access", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()

		invitation, err := f.svc.GetOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		require.Nil(t, invitation)
	})

	t.Run("get returns repository errors", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(nil, errRepo).Once()

		invitation, err := f.svc.GetOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitation)
	})

	t.Run("get hides invitations from another organization", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2"}, nil).Once()

		invitation, err := f.svc.GetOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, invitation)
	})

	t.Run("revoke rejects an empty invitation id", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "")

		require.ErrorIs(t, err, coreerrors.ErrUnauthorized)
		require.Nil(t, invitation)
	})

	t.Run("revoke rejects actors without organization access", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		require.Nil(t, invitation)
	})

	t.Run("revoke returns repository errors", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(nil, errRepo).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitation)
	})

	t.Run("revoke hides invitations from another organization", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2"}, nil).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, invitation)
	})

	t.Run("revoke returns expiration errors", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		expired := pendingInvitation("inv-1")
		expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitation)
	})

	t.Run("revoke rejects invitations that are no longer pending", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		revoked := pendingInvitation("inv-1")
		revoked.Status = orgtypes.OrganizationInvitationStatusRevoked
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(revoked, nil).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrConflict)
		require.Nil(t, invitation)
	})

	t.Run("revoke before update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("before revoke hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
		f.invRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("revoke after update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("after revoke hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectInvitationOrganizationAccess(f, "org-1", "user-1")
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := f.svc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor("user-1"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
		f.invRepo.AssertExpectations(t)
	})
}

func TestOrganizationInvitationService_AcceptAndRejectExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")

	expectAcceptUser := func(f *invitationExtraFixture) {
		f.userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
	}

	t.Run("accept returns expiration errors", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectAcceptUser(f)
		expired := pendingInvitation("inv-1")
		expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()

		invitation, err := f.svc.AcceptOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitation)
	})

	t.Run("accept before update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("before accept hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectAcceptUser(f)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := f.svc.AcceptOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
	})

	t.Run("accept reports a conflict when nothing was accepted", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectAcceptUser(f)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
		svc := newInvitationServiceForTest(f, invitationSkippingTxRunner{}, nil)

		invitation, err := svc.AcceptOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, coreerrors.ErrConflict)
		require.Nil(t, invitation)
	})

	t.Run("accept after update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("after accept hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectAcceptUser(f)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
		f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&orgtypes.OrganizationMember{ID: "mem-1"}, nil).Once()
		accepted := pendingInvitation("inv-1")
		accepted.Status = orgtypes.OrganizationInvitationStatusAccepted
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(accepted, nil).Once()

		invitation, err := f.svc.AcceptOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
	})

	t.Run("reject returns expiration errors", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		expectAcceptUser(f)
		expired := pendingInvitation("inv-1")
		expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()

		invitation, err := f.svc.RejectOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitation)
	})

	t.Run("reject before update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("before reject hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectAcceptUser(f)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := f.svc.RejectOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
	})

	t.Run("reject after update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		errHook := errors.New("after reject hook failed")
		executor := newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
			hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
				return errHook
			})
		})
		f := newInvitationExtraFixture(t, executor, nil)
		expectAcceptUser(f)
		f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
		f.invRepo.On("Update", mock.Anything, mock.Anything).Return(pendingInvitation("inv-1"), nil).Once()

		invitation, err := f.svc.RejectOrganizationInvitation(context.Background(), orgtests.Actor("user-2"), "org-1", "inv-1")

		require.ErrorIs(t, err, errHook)
		require.Nil(t, invitation)
	})
}

func TestOrganizationInvitationService_AcceptPendingExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")

	t.Run("missing user id is rejected", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		invitations, err := f.svc.AcceptPendingOrganizationInvitationsForEmail(context.Background(), "", "user@example.com")

		require.ErrorIs(t, err, coreerrors.ErrUnprocessableEntity)
		require.Nil(t, invitations)
	})

	t.Run("pending lookup error is returned", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		f.invRepo.On("GetAllPendingByEmail", mock.Anything, "user@example.com").Return(nil, errRepo).Once()

		invitations, err := f.svc.AcceptPendingOrganizationInvitationsForEmail(context.Background(), "user-2", "user@example.com")

		require.ErrorIs(t, err, errRepo)
		require.Nil(t, invitations)
	})

	t.Run("no pending invitations returns an empty slice", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		f.invRepo.On("GetAllPendingByEmail", mock.Anything, "user@example.com").Return(nil, nil).Once()

		invitations, err := f.svc.AcceptPendingOrganizationInvitationsForEmail(context.Background(), "user-2", "user@example.com")

		require.NoError(t, err)
		require.Empty(t, invitations)
	})
}

func TestOrganizationInvitationService_AcceptOrganizationInvitationsRoleErrors(t *testing.T) {
	t.Parallel()

	errRole := errors.New("role lookup failed")

	t.Run("role lookup error is returned", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		svc := newInvitationServiceWithOverrides(f, &orgtests.MockOrganizationInvitationTxRunner{}, nil, f.logger, &invitationRoleErrorAccessControl{
			AccessControlServiceStub: orgtests.NewAccessControlServiceStub(),
			err:                      errRole,
		})

		accepted, err := svc.acceptOrganizationInvitations(context.Background(), "user-2", []orgtypes.OrganizationInvitation{*pendingInvitation("inv-1")})

		require.ErrorIs(t, err, errRole)
		require.Nil(t, accepted)
	})

	t.Run("unknown role is rejected", func(t *testing.T) {
		t.Parallel()

		f := newInvitationExtraFixture(t, nil, nil)
		invitation := pendingInvitation("inv-1")
		invitation.Role = "ghost"

		accepted, err := f.svc.acceptOrganizationInvitations(context.Background(), "user-2", []orgtypes.OrganizationInvitation{*invitation})

		require.ErrorIs(t, err, coreerrors.ErrUnprocessableEntity)
		require.Nil(t, accepted)
	})
}
