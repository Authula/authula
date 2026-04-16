package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgevents "github.com/Authula/authula/plugins/organizations/events"
	"github.com/Authula/authula/plugins/organizations/repositories"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type testInvitationLogger struct {
	mu       sync.Mutex
	warnings []string
	errors   []string
}

func (l *testInvitationLogger) Debug(msg string, args ...any) {}
func (l *testInvitationLogger) Info(msg string, args ...any)  {}
func (l *testInvitationLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnings = append(l.warnings, msg)
}
func (l *testInvitationLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

func newTestOrganizationInvitationService(
	txRunner organizationInvitationTxRunner,
	pluginConfig *types.OrganizationsPluginConfig,
	userService rootservices.UserService,
	accessControlService rootservices.AccessControlService,
	orgRepo repositories.OrganizationRepository,
	invRepo repositories.OrganizationInvitationRepository,
	memberRepo repositories.OrganizationMemberRepository,
) *organizationInvitationService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	return NewOrganizationInvitationService(
		txRunner,
		&models.Config{BaseURL: "https://example.com", BasePath: "/auth"},
		pluginConfig,
		&testInvitationLogger{},
		nil,
		userService,
		nil,
		accessControlService,
		orgRepo,
		invRepo,
		memberRepo,
		serviceUtils,
	)
}

type invitationEmailCall struct {
	to      string
	subject string
	text    string
	html    string
}

type capturingMailer struct {
	called chan invitationEmailCall
	err    error
}

func (m *capturingMailer) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	if m.called != nil {
		m.called <- invitationEmailCall{to: to, subject: subject, text: text, html: html}
	}
	return m.err
}

type capturingEventBus struct {
	called chan models.Event
	err    error
}

func (b *capturingEventBus) Publish(ctx context.Context, event models.Event) error {
	if b.called != nil {
		b.called <- event
	}
	return b.err
}

func (b *capturingEventBus) Close() error { return nil }
func (b *capturingEventBus) Subscribe(topic string, handler models.EventHandler) (models.SubscriptionID, error) {
	return 0, nil
}
func (b *capturingEventBus) Unsubscribe(topic string, id models.SubscriptionID) {}

func TestOrganizationInvitationService_CreateOrganizationInvitation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		actorUserID          string
		organizationID       string
		request              types.CreateOrganizationInvitationRequest
		invitationExpiresIn  time.Duration
		accessControlService rootservices.AccessControlService
		setup                func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationInvitationHooks)
		expectErr            error
		expectCalled         bool
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			request:        types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "bad request empty email",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationInvitationRequest{Email: "", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrUnprocessableEntity,
		},
		{
			name:                "invalid role is rejected",
			actorUserID:         "user-1",
			organizationID:      "org-1",
			invitationExpiresIn: 36 * time.Hour,
			request:             types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "ghost"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr:    internalerrors.ErrUnprocessableEntity,
			expectCalled: true,
		},
		{
			name:                 "higher role is forbidden",
			actorUserID:          "user-1",
			organizationID:       "org-1",
			invitationExpiresIn:  36 * time.Hour,
			request:              types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "manager"},
			accessControlService: orgtests.NewAccessControlServiceStubWithWeights(nil, map[string]int{"user-1": 10}),
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr:    internalerrors.ErrForbidden,
			expectCalled: false,
		},
		{
			name:                 "access control forbidden is normalized",
			actorUserID:          "user-1",
			organizationID:       "org-1",
			invitationExpiresIn:  36 * time.Hour,
			request:              types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "manager"},
			accessControlService: &orgtests.AccessControlServiceStub{Err: errors.New("forbidden")},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr:    internalerrors.ErrForbidden,
			expectCalled: false,
		},
		{
			name:           "forbidden for non owner",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:                "org member can create",
			actorUserID:         "user-2",
			organizationID:      "org-1",
			invitationExpiresIn: time.Hour,
			request:             types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1", Name: "Acme"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.OrganizationID == "org-1" && invitation.InviterID == "user-2" && invitation.Email == "user@example.com" && invitation.Role == "member"
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-2", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectCalled: true,
		},
		{
			name:                "success",
			actorUserID:         "user-1",
			organizationID:      "org-1",
			invitationExpiresIn: 36 * time.Hour,
			request:             types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				expectedExpiresAt := time.Now().UTC().Add(36 * time.Hour)
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(inv *types.OrganizationInvitation) bool {
					return inv != nil && inv.OrganizationID == "org-1" && inv.InviterID == "user-1" && inv.Email == "user@example.com" && inv.Role == "member" && inv.Status == types.OrganizationInvitationStatusPending && inv.ExpiresAt.After(expectedExpiresAt.Add(-2*time.Second)) && inv.ExpiresAt.Before(expectedExpiresAt.Add(2*time.Second))
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: expectedExpiresAt}, nil).Once()
				hooks.Before = func(invitation *types.OrganizationInvitation) error {
					require.Equal(t, "user@example.com", invitation.Email)
					return nil
				}
				hooks.After = func(invitation types.OrganizationInvitation) error {
					require.Equal(t, "inv-1", invitation.ID)
					return nil
				}
			},
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:             true,
				InvitationExpiresIn: 24 * time.Hour,
			}
			if tt.invitationExpiresIn != 0 {
				pluginConfig.InvitationExpiresIn = tt.invitationExpiresIn
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			orgInvitationRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			orgInvitationHooks := &orgtests.MockOrganizationInvitationHooks{}
			if tt.setup != nil {
				tt.setup(orgRepo, orgInvitationRepo, memberRepo, orgInvitationHooks)
			}

			accessControlService := tt.accessControlService
			if accessControlService == nil {
				accessControlService = orgtests.NewAccessControlServiceStub()
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, accessControlService, orgRepo, orgInvitationRepo, memberRepo)
			inv, err := svc.CreateOrganizationInvitation(context.Background(), tt.actorUserID, tt.organizationID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				if strings.Contains(tt.name, "invalid role") {
					require.ErrorContains(t, err, tt.expectErr.Error())
				} else {
					require.ErrorIs(t, err, tt.expectErr)
				}
				require.True(t, orgInvitationRepo.AssertExpectations(t))
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, inv)
			require.WithinDuration(t, time.Now().UTC().Add(tt.invitationExpiresIn), inv.ExpiresAt, 2*time.Second)
			require.True(t, orgInvitationRepo.AssertExpectations(t))
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_GetOrganizationInvitation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		invitationID   string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectID       string
		expectStatus   types.OrganizationInvitationStatus
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectID:     "inv-1",
			expectStatus: types.OrganizationInvitationStatusPending,
		},
		{
			name:           "rejected invitation is returned",
			actorUserID:    "user-1",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusRejected, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectID:     "inv-1",
			expectStatus: types.OrganizationInvitationStatusRejected,
		},
		{
			name:           "org member can get",
			actorUserID:    "user-2",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectID:     "inv-1",
			expectStatus: types.OrganizationInvitationStatusPending,
		},
		{
			name:           "pending invitation is returned as pending",
			actorUserID:    "user-2",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)}, nil).Once()
			},
			expectID:     "inv-1",
			expectStatus: types.OrganizationInvitationStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:             true,
				InvitationExpiresIn: 24 * time.Hour,
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo, memberRepo)
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitation, err := svc.GetOrganizationInvitation(context.Background(), tt.actorUserID, tt.organizationID, tt.invitationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectID, invitation.ID)
			require.Equal(t, tt.expectStatus, invitation.Status)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, invRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_GetAllOrganizationInvitations(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectLen      int
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return([]types.OrganizationInvitation{{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil).Once()
			},
			expectLen: 1,
		},
		{
			name:           "rejected invitation is returned",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return([]types.OrganizationInvitation{{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusRejected, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil).Once()
			},
			expectLen: 1,
		},
		{
			name:           "org member can list",
			actorUserID:    "user-2",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return([]types.OrganizationInvitation{{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil).Once()
			},
			expectLen: 1,
		},
		{name: "unauthorized", actorUserID: "", organizationID: "org-1", expectErr: internalerrors.ErrUnauthorized},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "repo error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return(([]types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:             true,
				InvitationExpiresIn: 24 * time.Hour,
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo, memberRepo)
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitations, err := svc.GetAllOrganizationInvitations(context.Background(), tt.actorUserID, tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, invRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.Len(t, invitations, tt.expectLen)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, invRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_RevokeOrganizationInvitation(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		invitationID   string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationInvitationHooks)
		expectErr      error
		expectStatus   types.OrganizationInvitationStatus
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusRevoked
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusRevoked}, nil).Once()
			},
			expectStatus: types.OrganizationInvitationStatusRevoked,
		},
		{
			name:           "org member can revoke",
			actorUserID:    "user-2",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusRevoked
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusRevoked}, nil).Once()
			},
			expectStatus: types.OrganizationInvitationStatusRevoked,
		},
		{
			name:           "expired pending conflict",
			actorUserID:    "user-1",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusExpired
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusExpired}, nil).Once()
			},
			expectErr: internalerrors.ErrConflict,
		},
		{
			name:           "update error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			invitationID:   "inv-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusRevoked
				})).Return((*types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:             true,
				InvitationExpiresIn: 24 * time.Hour,
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationInvitationHooks{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo, memberRepo, hooks)
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitation, err := svc.RevokeOrganizationInvitation(context.Background(), tt.actorUserID, tt.organizationID, tt.invitationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, invRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectStatus, invitation.Status)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, invRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_AcceptPendingOrganizationInvitationsForEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		userID                   string
		email                    string
		requireEmailVerification bool
		setup                    func(*internaltests.MockUserService, *orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationInvitationHooks, *orgtests.MockOrganizationMemberHooks)
		expectErr                error
		expectLen                int
	}{
		{
			name:      "bad request invalid email",
			userID:    "user-2",
			email:     "not-an-email",
			expectErr: internalerrors.ErrBadRequest,
		},
		{
			name:   "success",
			userID: "user-2",
			email:  "USER@EXAMPLE.COM",
			setup: func(userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				invRepo.On("GetAllPendingByEmail", mock.Anything, "user@example.com").Return([]types.OrganizationInvitation{{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2" && member.Role == "member"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.ID == "inv-1" && invitation.Status == types.OrganizationInvitationStatusAccepted
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusAccepted}, nil).Once()
			},
			expectLen: 1,
		},
		{
			name:                     "forbidden when email verification is required and user is unverified",
			userID:                   "user-2",
			email:                    "USER@EXAMPLE.COM",
			requireEmailVerification: true,
			setup: func(userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com", EmailVerified: false}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:                          true,
				InvitationExpiresIn:              24 * time.Hour,
				RequireEmailVerifiedOnInvitation: tt.requireEmailVerification,
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationInvitationHooks{}
			memberHooks := &orgtests.MockOrganizationMemberHooks{}
			userSvc := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(userSvc, orgRepo, invRepo, memberRepo, hooks, memberHooks)
			}

			txRunner := &orgtests.MockOrganizationInvitationTxRunner{}
			svc := newTestOrganizationInvitationService(txRunner, pluginConfig, userSvc, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			accepted, err := svc.AcceptPendingOrganizationInvitationsForEmail(context.Background(), tt.userID, tt.email)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectErr.Error())
				return
			}
			require.NoError(t, err)
			require.Len(t, accepted, tt.expectLen)
		})
	}
}

func TestOrganizationInvitationService_AcceptOrganizationInvitation(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name                     string
		actorUserID              string
		organization             string
		invitationID             string
		requireEmailVerification bool
		setup                    func(*internaltests.MockUserService, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationInvitationHooks, *orgtests.MockOrganizationMemberHooks)
		expectErr                error
		expectStatus             types.OrganizationInvitationStatus
	}{
		{
			name:         "unauthorized",
			actorUserID:  "",
			organization: "org-1",
			invitationID: "inv-1",
			expectErr:    internalerrors.ErrUnauthorized,
		},
		{
			name:         "success",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationMember")).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.ID == "inv-1" && invitation.Status == types.OrganizationInvitationStatusAccepted
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusAccepted}, nil).Once()
			},
			expectStatus: types.OrganizationInvitationStatusAccepted,
		},
		{
			name:         "forbidden when emails differ",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "other@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:         "user lookup error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return((*models.User)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "user not found",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return((*models.User)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "user missing email",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: ""}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "invitation lookup error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return((*types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "invitation not found",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return((*types.OrganizationInvitation)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "invitation from other organization",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "expired pending conflict",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusExpired
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusExpired}, nil).Once()
			},
			expectErr: internalerrors.ErrConflict,
		},
		{
			name:         "member lookup error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return((*types.OrganizationMember)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "member create error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2"
				})).Return((*types.OrganizationMember)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "update error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusAccepted
				})).Return((*types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:                     "forbidden when email verification is required and user is unverified",
			actorUserID:              "user-2",
			organization:             "org-1",
			invitationID:             "inv-1",
			requireEmailVerification: true,
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationInvitationHooks, memberHooks *orgtests.MockOrganizationMemberHooks) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com", EmailVerified: false}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:                          true,
				InvitationExpiresIn:              24 * time.Hour,
				RequireEmailVerifiedOnInvitation: tt.requireEmailVerification,
			}
			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationInvitationHooks{}
			memberHooks := &orgtests.MockOrganizationMemberHooks{}
			userSvc := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(userSvc, invRepo, memberRepo, hooks, memberHooks)
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, userSvc, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitation, err := svc.AcceptOrganizationInvitation(context.Background(), tt.actorUserID, tt.organization, tt.invitationID)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectStatus, invitation.Status)
		})
	}
}

func TestOrganizationInvitationService_RejectOrganizationInvitation(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name         string
		actorUserID  string
		organization string
		invitationID string
		setup        func(*internaltests.MockUserService, *orgtests.MockOrganizationInvitationRepository)
		expectErr    error
		expectStatus types.OrganizationInvitationStatus
	}{
		{
			name:         "unauthorized",
			actorUserID:  "",
			organization: "org-1",
			invitationID: "inv-1",
			expectErr:    internalerrors.ErrUnauthorized,
		},
		{
			name:         "success",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusRejected
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusRejected}, nil).Once()
			},
			expectStatus: types.OrganizationInvitationStatusRejected,
		},
		{
			name:         "user lookup error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return((*models.User)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "user not found",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return((*models.User)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "user missing email",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: ""}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "invitation lookup error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return((*types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "invitation not found",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return((*types.OrganizationInvitation)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "invitation from other organization",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "expired pending conflict",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusExpired
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusExpired}, nil).Once()
			},
			expectErr: internalerrors.ErrConflict,
		},
		{
			name:         "email mismatch forbidden",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "other@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:         "update error",
			actorUserID:  "user-2",
			organization: "org-1",
			invitationID: "inv-1",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.Status == types.OrganizationInvitationStatusRejected
				})).Return((*types.OrganizationInvitation)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := &types.OrganizationsPluginConfig{
				Enabled:             true,
				InvitationExpiresIn: 24 * time.Hour,
			}
			userSvc := &internaltests.MockUserService{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			if tt.setup != nil {
				tt.setup(userSvc, invRepo)
			}

			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, userSvc, orgtests.NewAccessControlServiceStub(), &orgtests.MockOrganizationRepository{}, invRepo, &orgtests.MockOrganizationMemberRepository{})
			invitation, err := svc.RejectOrganizationInvitation(context.Background(), tt.actorUserID, tt.organization, tt.invitationID)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectStatus, invitation.Status)
		})
	}
}

func TestOrganizationInvitationService_CreateOrganizationInvitation_SideEffects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		actorUserID     string
		organizationID  string
		request         types.CreateOrganizationInvitationRequest
		setup           func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository)
		mailerFactory   func() (*capturingMailer, chan invitationEmailCall)
		eventBusFactory func() (*capturingEventBus, chan models.Event)
		verify          func(*testing.T, *types.OrganizationInvitation, *testInvitationLogger, chan invitationEmailCall, chan models.Event)
	}{
		{
			name:           "sends email and publishes event",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member", RedirectURL: "https://app.example.com/welcome"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(inv *types.OrganizationInvitation) bool {
					return inv != nil && inv.OrganizationID == "org-1" && inv.InviterID == "user-1" && inv.Email == "user@example.com" && inv.Role == "member" && inv.Status == types.OrganizationInvitationStatusPending
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(36 * time.Hour)}, nil).Once()
			},
			mailerFactory: func() (*capturingMailer, chan invitationEmailCall) {
				mailerCalls := make(chan invitationEmailCall, 1)
				return &capturingMailer{called: mailerCalls}, mailerCalls
			},
			eventBusFactory: func() (*capturingEventBus, chan models.Event) {
				eventCalls := make(chan models.Event, 1)
				return &capturingEventBus{called: eventCalls}, eventCalls
			},
			verify: func(t *testing.T, invitation *types.OrganizationInvitation, logger *testInvitationLogger, mailerCalls chan invitationEmailCall, eventCalls chan models.Event) {
				require.NotNil(t, invitation)
				require.Eventually(t, func() bool { return len(mailerCalls) > 0 }, time.Second, 10*time.Millisecond)
				require.Eventually(t, func() bool { return len(eventCalls) > 0 }, time.Second, 10*time.Millisecond)

				mailCall := <-mailerCalls
				require.Equal(t, "user@example.com", mailCall.to)
				require.Contains(t, mailCall.text, "https://example.com/auth/organizations/org-1/invitations/inv-1/accept?redirect_url=https%3A%2F%2Fapp.example.com%2Fwelcome")
				require.Contains(t, mailCall.html, "Accept invitation")

				event := <-eventCalls
				require.Equal(t, orgconstants.EventOrganizationsInvitationCreated, event.Type)

				var payload orgevents.OrganizationInvitationCreatedEvent
				require.NoError(t, json.Unmarshal(event.Payload, &payload))
				require.Equal(t, "inv-1", payload.InvitationID)
				require.Equal(t, "org-1", payload.OrganizationID)
				require.Equal(t, "Acme", payload.OrganizationName)
				require.Equal(t, "user@example.com", payload.InviteeEmail)
				require.Empty(t, logger.warnings)
			},
		},
		{
			name:           "skips missing mailer",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(36 * time.Hour)}, nil).Once()
			},
			verify: func(t *testing.T, invitation *types.OrganizationInvitation, logger *testInvitationLogger, _ chan invitationEmailCall, _ chan models.Event) {
				require.NotNil(t, invitation)
				require.Empty(t, logger.errors)
				require.Contains(t, logger.warnings, "mailer service not available, skipping organization invitation email")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo)
			}

			logger := &testInvitationLogger{}
			var mailer rootservices.MailerService
			var mailerCalls chan invitationEmailCall
			if tt.mailerFactory != nil {
				mailerImpl, calls := tt.mailerFactory()
				mailer = mailerImpl
				mailerCalls = calls
			}
			var eventBus models.EventBus
			var eventCalls chan models.Event
			if tt.eventBusFactory != nil {
				eventBusImpl, calls := tt.eventBusFactory()
				eventBus = eventBusImpl
				eventCalls = calls
			}

			serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}

			svc := NewOrganizationInvitationService(
				&orgtests.MockOrganizationInvitationTxRunner{},
				&models.Config{BaseURL: "https://example.com", BasePath: "/auth"},
				&types.OrganizationsPluginConfig{Enabled: true, InvitationExpiresIn: 36 * time.Hour},
				logger,
				eventBus,
				&internaltests.MockUserService{},
				mailer,
				orgtests.NewAccessControlServiceStub(),
				orgRepo,
				invRepo,
				memberRepo,
				serviceUtils,
			)

			invitation, err := svc.CreateOrganizationInvitation(context.Background(), tt.actorUserID, tt.organizationID, tt.request)
			require.NoError(t, err)
			require.NotNil(t, invitation)
			if tt.verify != nil {
				tt.verify(t, invitation, logger, mailerCalls, eventCalls)
			}
		})
	}
}

func TestOrganizationInvitationService_CreateOrganizationInvitation_MembersLimit(t *testing.T) {
	t.Parallel()

	zeroLimit := 0
	threeLimit := 3
	twoLimit := 2

	tests := []struct {
		name            string
		membersLimit    *int
		setup           func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr       error
		expectCreatedID string
	}{
		{
			name:         "zero limit treated as unlimited",
			membersLimit: &zeroLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.OrganizationID == "org-1" && invitation.InviterID == "user-1" && invitation.Email == "user@example.com" && invitation.Role == "member"
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectCreatedID: "inv-1",
		},
		{
			name:         "success within limit",
			membersLimit: &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.OrganizationID == "org-1" && invitation.InviterID == "user-1" && invitation.Email == "user@example.com" && invitation.Role == "member"
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectCreatedID: "inv-1",
		},
		{
			name:         "quota exceeded",
			membersLimit: &twoLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
			},
			expectErr: orgconstants.ErrMembersQuotaExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo, memberRepo)
			}

			pluginConfig := &types.OrganizationsPluginConfig{Enabled: true, InvitationExpiresIn: time.Hour, MembersLimit: tt.membersLimit}
			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitation, err := svc.CreateOrganizationInvitation(context.Background(), "user-1", "org-1", types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"})
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectCreatedID, invitation.ID)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, invRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_CreateOrganizationInvitation_InvitationsLimit(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")
	zeroLimit := 0
	threeLimit := 3
	twoLimit := 2

	tests := []struct {
		name             string
		membersLimit     *int
		invitationsLimit *int
		setup            func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr        error
	}{
		{
			name:             "member quota still blocks first",
			membersLimit:     &twoLimit,
			invitationsLimit: &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
			},
			expectErr: orgconstants.ErrMembersQuotaExceeded,
		},
		{
			name:             "zero invitations limit treated as unlimited",
			invitationsLimit: &zeroLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.OrganizationID == "org-1" && invitation.InviterID == "user-1" && invitation.Email == "user@example.com" && invitation.Role == "member"
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
		},
		{
			name:             "success within invitations limit",
			invitationsLimit: &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(2, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				invRepo.On("Create", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.OrganizationID == "org-1" && invitation.InviterID == "user-1" && invitation.Email == "user@example.com" && invitation.Role == "member"
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", InviterID: "user-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
		},
		{
			name:             "quota exceeded",
			invitationsLimit: &twoLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(2, nil).Once()
			},
			expectErr: orgconstants.ErrInvitationsQuotaExceeded,
		},
		{
			name:             "pending invitation still conflicts",
			invitationsLimit: &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(1, nil).Once()
				invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com", types.OrganizationInvitationStatusPending).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
			},
			expectErr: internalerrors.ErrConflict,
		},
		{
			name:             "repository count error",
			invitationsLimit: &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(0, someErr).Once()
			},
			expectErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, invRepo, memberRepo)
			}

			pluginConfig := &types.OrganizationsPluginConfig{Enabled: true, InvitationExpiresIn: time.Hour, MembersLimit: tt.membersLimit, InvitationsLimit: tt.invitationsLimit}
			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, &internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, invRepo, memberRepo)
			invitation, err := svc.CreateOrganizationInvitation(context.Background(), "user-1", "org-1", types.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"})
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, invitation)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, invRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationInvitationService_AcceptOrganizationInvitation_MembersLimit(t *testing.T) {
	t.Parallel()

	limit := 2

	tests := []struct {
		name         string
		setup        func(*internaltests.MockUserService, *orgtests.MockOrganizationInvitationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr    error
		expectStatus types.OrganizationInvitationStatus
	}{
		{
			name: "quota exceeded",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
			},
			expectErr: orgconstants.ErrMembersQuotaExceeded,
		},
		{
			name: "existing member is still accepted at capacity",
			setup: func(userSvc *internaltests.MockUserService, invRepo *orgtests.MockOrganizationInvitationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user@example.com"}, nil).Once()
				invRepo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				invRepo.On("Update", mock.Anything, mock.MatchedBy(func(invitation *types.OrganizationInvitation) bool {
					return invitation != nil && invitation.ID == "inv-1" && invitation.Status == types.OrganizationInvitationStatusAccepted
				})).Return(&types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: types.OrganizationInvitationStatusAccepted}, nil).Once()
			},
			expectStatus: types.OrganizationInvitationStatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userSvc := &internaltests.MockUserService{}
			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(userSvc, invRepo, memberRepo)
			}

			pluginConfig := &types.OrganizationsPluginConfig{Enabled: true, InvitationExpiresIn: time.Hour, MembersLimit: &limit}
			svc := newTestOrganizationInvitationService(&orgtests.MockOrganizationInvitationTxRunner{}, pluginConfig, userSvc, orgtests.NewAccessControlServiceStub(), &orgtests.MockOrganizationRepository{}, invRepo, memberRepo)
			invitation, err := svc.AcceptOrganizationInvitation(context.Background(), "user-2", "org-1", "inv-1")
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, invitation)
			require.Equal(t, tt.expectStatus, invitation.Status)
		})
	}
}
