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

func expectInvitationAcceptUser(f *invitationExtraFixture, userID, email string) {
	f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: email}, nil).Once()
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

	tests := []struct {
		name            string
		signalingLogger *invitationSignalingLogger
		setup           func(*invitationExtraFixture)
		expectErr       error
		expectLog       string
	}{
		{
			name: "existing invitation lookup error is returned",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", orgtypes.OrganizationInvitationStatusPending).
					Return(nil, errLookup).Once()
			},
			expectErr: errLookup,
		},
		{
			name:            "background email error is logged",
			signalingLogger: newInvitationSignalingLogger(),
			setup: func(f *invitationExtraFixture) {
				f.mailer = &capturingMailer{err: errMail}
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", orgtypes.OrganizationInvitationStatusPending).
					Return(nil, nil).Once()
				f.invRepo.On("Create", mock.Anything, mock.Anything).
					Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectLog: "failed to send organization invitation email via built-in email service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, nil, func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				if tt.setup != nil {
					tt.setup(f)
				}
			})

			logger := models.Logger(f.logger)
			if tt.signalingLogger != nil {
				logger = tt.signalingLogger
			}
			svc := newInvitationServiceWithOverrides(f, &orgtests.MockOrganizationInvitationTxRunner{}, nil, logger, f.accessControl)

			invitation, err := svc.CreateOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-1"),
				"org-1",
				orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
				"",
			)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, invitation)
			} else {
				require.NoError(t, err)
				require.NotNil(t, invitation)
			}

			if tt.expectLog != "" {
				require.NotNil(t, tt.signalingLogger)
				waitForInvitationLog(t, tt.signalingLogger.done, "background email error")
				require.Contains(t, strings.Join(tt.signalingLogger.messages(), "\n"), tt.expectLog)
			}

			require.True(t, f.orgRepo.AssertExpectations(t))
			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.memberRepo.AssertExpectations(t))
			require.True(t, f.userSvc.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_GetOrganizationInvitationExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")

	tests := []struct {
		name         string
		invitationID string
		setup        func(*invitationExtraFixture)
		expectErr    error
	}{
		{
			name:         "empty invitation id is rejected",
			invitationID: "",
			expectErr:    coreerrors.ErrUnauthorized,
		},
		{
			name:         "actor without organization access is rejected",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").
					Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:         "repository errors are returned",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name:         "invitations from another organization are hidden",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").
					Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, nil, tt.setup)

			invitation, err := f.svc.GetOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-1"),
				"org-1",
				tt.invitationID,
			)

			require.ErrorIs(t, err, tt.expectErr)
			require.Nil(t, invitation)

			require.True(t, f.orgRepo.AssertExpectations(t))
			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_RevokeOrganizationInvitationExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")
	errHook := errors.New("hook failed")

	tests := []struct {
		name                string
		invitationID        string
		hooks               *ServiceHookExecutor
		setup               func(*invitationExtraFixture)
		expectErr           error
		assertUpdateSkipped bool
	}{
		{
			name:         "empty invitation id is rejected",
			invitationID: "",
			expectErr:    coreerrors.ErrUnauthorized,
		},
		{
			name:         "actor without organization access is rejected",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").
					Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:         "repository errors are returned",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name:         "invitations from another organization are hidden",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").
					Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:         "expiration errors are returned",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				expired := pendingInvitation("inv-1")
				expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name:         "invitations that are no longer pending are rejected",
			invitationID: "inv-1",
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				revoked := pendingInvitation("inv-1")
				revoked.Status = orgtypes.OrganizationInvitationStatusRevoked
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(revoked, nil).Once()
			},
			expectErr: coreerrors.ErrConflict,
		},
		{
			name:         "before update hook error is propagated",
			invitationID: "inv-1",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr:           errHook,
			assertUpdateSkipped: true,
		},
		{
			name:         "after update hook error is propagated",
			invitationID: "inv-1",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				expectInvitationOrganizationAccess(f, "org-1", "user-1")
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr: errHook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, tt.hooks, tt.setup)

			invitation, err := f.svc.RevokeOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-1"),
				"org-1",
				tt.invitationID,
			)

			require.ErrorIs(t, err, tt.expectErr)
			require.Nil(t, invitation)

			if tt.assertUpdateSkipped {
				f.invRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			}

			require.True(t, f.orgRepo.AssertExpectations(t))
			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_AcceptOrganizationInvitationExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")
	errHook := errors.New("hook failed")

	tests := []struct {
		name      string
		hooks     *ServiceHookExecutor
		txRunner  organizationInvitationTxRunner
		setup     func(*invitationExtraFixture)
		expectErr error
	}{
		{
			name: "expiration errors are returned",
			setup: func(f *invitationExtraFixture) {
				expired := pendingInvitation("inv-1")
				expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name: "before update hook error is propagated",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr: errHook,
		},
		{
			name:     "a conflict is reported when nothing was accepted",
			txRunner: invitationSkippingTxRunner{},
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr: coreerrors.ErrConflict,
		},
		{
			name: "after update hook error is propagated",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
				f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").
					Return(&orgtypes.OrganizationMember{ID: "mem-1"}, nil).Once()
				accepted := pendingInvitation("inv-1")
				accepted.Status = orgtypes.OrganizationInvitationStatusAccepted
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(accepted, nil).Once()
			},
			expectErr: errHook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, tt.hooks, tt.setup)
			expectInvitationAcceptUser(f, "user-2", "user@example.com")

			txRunner := organizationInvitationTxRunner(&orgtests.MockOrganizationInvitationTxRunner{})
			if tt.txRunner != nil {
				txRunner = tt.txRunner
			}
			svc := newInvitationServiceForTest(f, txRunner, tt.hooks)

			invitation, err := svc.AcceptOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-2"),
				"org-1",
				"inv-1",
			)

			require.ErrorIs(t, err, tt.expectErr)
			require.Nil(t, invitation)

			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.memberRepo.AssertExpectations(t))
			require.True(t, f.userSvc.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_RejectOrganizationInvitationExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")
	errHook := errors.New("hook failed")

	tests := []struct {
		name      string
		hooks     *ServiceHookExecutor
		setup     func(*invitationExtraFixture)
		expectErr error
	}{
		{
			name: "expiration errors are returned",
			setup: func(f *invitationExtraFixture) {
				expired := pendingInvitation("inv-1")
				expired.ExpiresAt = time.Now().UTC().Add(-time.Hour)
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(expired, nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name: "before update hook error is propagated",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr: errHook,
		},
		{
			name: "after update hook error is propagated",
			hooks: newInvitationHooksExecutor(func(hooks *orgtypes.OrganizationInvitationServiceHooks) {
				hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationInvitation) error {
					return errHook
				})
			}),
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByID", mock.Anything, "inv-1").Return(pendingInvitation("inv-1"), nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.Anything).Return(pendingInvitation("inv-1"), nil).Once()
			},
			expectErr: errHook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, tt.hooks, tt.setup)
			expectInvitationAcceptUser(f, "user-2", "user@example.com")

			svc := newInvitationServiceForTest(f, &orgtests.MockOrganizationInvitationTxRunner{}, tt.hooks)

			invitation, err := svc.RejectOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-2"),
				"org-1",
				"inv-1",
			)

			require.ErrorIs(t, err, tt.expectErr)
			require.Nil(t, invitation)

			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.userSvc.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_AcceptPendingExtraBranches(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repository unavailable")

	tests := []struct {
		name        string
		userID      string
		email       string
		setup       func(*invitationExtraFixture)
		expectErr   error
		expectEmpty bool
	}{
		{
			name:      "missing user id is rejected",
			userID:    "",
			email:     "user@example.com",
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:   "pending lookup error is returned",
			userID: "user-2",
			email:  "user@example.com",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetAllPendingByEmail", mock.Anything, "user@example.com").Return(nil, errRepo).Once()
			},
			expectErr: errRepo,
		},
		{
			name:   "no pending invitations returns an empty slice",
			userID: "user-2",
			email:  "user@example.com",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetAllPendingByEmail", mock.Anything, "user@example.com").Return(nil, nil).Once()
			},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, nil, tt.setup)

			invitations, err := f.svc.AcceptPendingOrganizationInvitationsForEmail(
				context.Background(),
				tt.userID,
				tt.email,
			)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, invitations)
			} else {
				require.NoError(t, err)
				require.Empty(t, invitations)
			}

			require.True(t, f.invRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_AcceptOrganizationInvitationsRoleErrors(t *testing.T) {
	t.Parallel()

	errRole := errors.New("role lookup failed")

	tests := []struct {
		name          string
		accessControl func(*invitationExtraFixture) rootservices.AccessControlService
		role          string
		expectErr     error
	}{
		{
			name: "role lookup error is returned",
			accessControl: func(*invitationExtraFixture) rootservices.AccessControlService {
				return &invitationRoleErrorAccessControl{
					AccessControlServiceStub: orgtests.NewAccessControlServiceStub(),
					err:                      errRole,
				}
			},
			role:      "member",
			expectErr: errRole,
		},
		{
			name:      "unknown role is rejected",
			role:      "ghost",
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newInvitationExtraFixture(t, nil, nil)
			var accessControl rootservices.AccessControlService = f.accessControl
			if tt.accessControl != nil {
				accessControl = tt.accessControl(f)
			}
			svc := newInvitationServiceWithOverrides(
				f,
				&orgtests.MockOrganizationInvitationTxRunner{},
				nil,
				f.logger,
				accessControl,
			)

			invitation := pendingInvitation("inv-1")
			invitation.Role = tt.role

			accepted, err := svc.acceptOrganizationInvitations(
				context.Background(),
				"user-2",
				[]orgtypes.OrganizationInvitation{*invitation},
			)

			require.ErrorIs(t, err, tt.expectErr)
			require.Nil(t, accepted)

			require.True(t, f.invRepo.AssertExpectations(t))
			require.True(t, f.memberRepo.AssertExpectations(t))
		})
	}
}
