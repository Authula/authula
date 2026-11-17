package services

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	emailtmpl "github.com/Authula/authula/core/email/template"
	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

type invitationExtraFixture struct {
	svc           *organizationInvitationService
	orgRepo       *orgtests.MockOrganizationRepository
	invRepo       *orgtests.MockOrganizationInvitationRepository
	memberRepo    *orgtests.MockOrganizationMemberRepository
	userSvc       *internaltests.MockUserService
	mailer        *capturingMailer
	accessControl *orgtests.AccessControlServiceStub
	pluginConfig  *types.OrganizationsPluginConfig
	logger        *testInvitationLogger
	templateMgr   *emailtmpl.Manager
}

func newInvitationExtraFixture(t *testing.T, hooks *ServiceHookExecutor, configure func(*invitationExtraFixture)) *invitationExtraFixture {
	t.Helper()

	templateMgr := emailtmpl.NewManager()
	require.NoError(t, templateMgr.Register(emailtmpl.Definition{
		Name:    "organization_invitation",
		Subject: "You're invited to join {{.OrganizationName}} on {{.AppName}}",
		Text:    "Join {{.OrganizationName}} as {{.Role}} via {{.InviteLink}}",
		HTML:    "<p>Join {{.OrganizationName}} as {{.Role}}</p>",
	}))

	fixture := &invitationExtraFixture{
		orgRepo:       &orgtests.MockOrganizationRepository{},
		invRepo:       &orgtests.MockOrganizationInvitationRepository{},
		memberRepo:    &orgtests.MockOrganizationMemberRepository{},
		userSvc:       &internaltests.MockUserService{},
		mailer:        &capturingMailer{},
		accessControl: orgtests.NewAccessControlServiceStub(),
		pluginConfig:  &types.OrganizationsPluginConfig{InvitationExpiresIn: time.Hour},
		logger:        &testInvitationLogger{},
		templateMgr:   templateMgr,
	}
	if configure != nil {
		configure(fixture)
	}

	fixture.svc = NewOrganizationInvitationService(
		&orgtests.MockOrganizationInvitationTxRunner{},
		&models.Config{BaseURL: "https://example.com", BasePath: "/auth"},
		fixture.pluginConfig,
		fixture.logger,
		nil,
		fixture.userSvc,
		fixture.mailer,
		fixture.accessControl,
		fixture.orgRepo,
		fixture.invRepo,
		fixture.memberRepo,
		&ServiceUtils{orgRepo: fixture.orgRepo, orgMemberRepo: fixture.memberRepo},
		fixture.templateMgr,
		hooks,
	)
	return fixture
}

func TestOrganizationInvitationService_GetOrganizationInvitationByID(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name         string
		invitationID string
		setup        func(*orgtests.MockOrganizationInvitationRepository)
		expectErr    error
		expectID     string
	}{
		{
			name:         "empty id is not found without hitting the repository",
			invitationID: "",
			expectErr:    coreerrors.ErrNotFound,
		},
		{
			name:         "repository error is propagated",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByID", mock.Anything, "inv-1").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "missing invitation is not found",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByID", mock.Anything, "inv-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:         "invitation is returned",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByID", mock.Anything, "inv-1").Return(&types.OrganizationInvitation{ID: "inv-1"}, nil).Once()
			},
			expectID: "inv-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newInvitationExtraFixture(t, nil, nil)
			if tt.setup != nil {
				tt.setup(fixture.invRepo)
			}

			invitation, err := fixture.svc.GetOrganizationInvitationByID(context.Background(), tt.invitationID)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, invitation)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectID, invitation.ID)
			}
			fixture.invRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationInvitationService_GetOrganizationInvitationByIDWithOrg(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")
	response := &types.GetOrganizationInvitationResponse{
		Invitation:   &types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1"},
		Organization: types.OrganizationSummary{ID: "org-1", Name: "Acme"},
	}

	tests := []struct {
		name         string
		invitationID string
		setup        func(*orgtests.MockOrganizationInvitationRepository)
		expectErr    error
	}{
		{
			name:         "empty id is not found without hitting the repository",
			invitationID: "",
			expectErr:    coreerrors.ErrNotFound,
		},
		{
			name:         "repository error is propagated",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByIDWithOrg", mock.Anything, "inv-1").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:         "missing invitation is not found",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByIDWithOrg", mock.Anything, "inv-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:         "response is returned",
			invitationID: "inv-1",
			setup: func(repo *orgtests.MockOrganizationInvitationRepository) {
				repo.On("GetByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newInvitationExtraFixture(t, nil, nil)
			if tt.setup != nil {
				tt.setup(fixture.invRepo)
			}

			got, err := fixture.svc.GetOrganizationInvitationByIDWithOrg(context.Background(), tt.invitationID)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, response, got)
			}
			fixture.invRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationInvitationService_CreateGuardRails(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook failure")
	repoErr := errors.New("repository error")
	now := time.Now().UTC()
	created := &types.OrganizationInvitation{
		ID:             "inv-1",
		Email:          "bob@example.com",
		InviterID:      "user-1",
		OrganizationID: "org-1",
		Role:           "member",
		Status:         types.OrganizationInvitationStatusPending,
		ExpiresAt:      now.Add(time.Hour),
	}

	tests := []struct {
		name      string
		setup     func(*invitationExtraFixture)
		hooks     *ServiceHookExecutor
		request   types.CreateOrganizationInvitationRequest
		expectErr error
		expectID  string
	}{
		{
			name: "an unknown empty role is still rejected",
			setup: func(f *invitationExtraFixture) {
				f.accessControl.RoleWeights[""] = 0
			},
			request:   types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: ""},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "a non-positive expiry is rejected",
			setup: func(f *invitationExtraFixture) {
				f.pluginConfig.InvitationExpiresIn = -time.Hour
			},
			request:   types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:  "a before-create hook error aborts the transaction",
			hooks: hookErrorExecutor(t, hookErr),
			request: types.CreateOrganizationInvitationRequest{
				Email: "bob@example.com", Role: "member",
			},
			expectErr: hookErr,
		},
		{
			name: "an after-create hook error is returned after the row is written",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "bob@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				f.invRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(created, nil).Once()
			},
			hooks: hookAfterCreateErrorExecutor(t, hookErr),
			request: types.CreateOrganizationInvitationRequest{
				Email: "bob@example.com", Role: "member",
			},
			expectErr: hookErr,
		},
		{
			name: "an expired pending invitation update error is returned",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "bob@example.com", types.OrganizationInvitationStatusPending).
					Return(&types.OrganizationInvitation{ID: "inv-old", OrganizationID: "org-1", Email: "bob@example.com", Status: types.OrganizationInvitationStatusPending, ExpiresAt: now.Add(-time.Hour)}, nil).Once()
				f.invRepo.On("Update", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(nil, repoErr).Once()
			},
			request:   types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"},
			expectErr: repoErr,
		},
		{
			name: "a create repository error is returned",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "bob@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				f.invRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(nil, repoErr).Once()
			},
			request:   types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"},
			expectErr: repoErr,
		},
		{
			name: "an inviter lookup error is returned when a callback sends the email",
			setup: func(f *invitationExtraFixture) {
				f.pluginConfig.SendOrganizationInvitationEmail = func(types.SendOrganizationInvitationEmailParams, *models.RequestContext) error {
					return nil
				}
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "bob@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				f.invRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(created, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, repoErr).Once()
			},
			request:   types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"},
			expectErr: repoErr,
		},
		{
			name: "the invitation is created when every guard passes",
			setup: func(f *invitationExtraFixture) {
				f.invRepo.On("GetByOrganizationIDAndEmail", mock.Anything, "org-1", "bob@example.com", types.OrganizationInvitationStatusPending).Return(nil, nil).Once()
				f.invRepo.On("Create", mock.Anything, mock.AnythingOfType("*types.OrganizationInvitation")).Return(created, nil).Once()
			},
			request:  types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"},
			expectID: "inv-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newInvitationExtraFixture(t, tt.hooks, func(f *invitationExtraFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme"}, nil).Once()
				f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").
					Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
				if tt.setup != nil {
					tt.setup(f)
				}
			})

			invitation, err := fixture.svc.CreateOrganizationInvitation(
				context.Background(),
				orgtests.Actor("user-1"),
				"org-1",
				tt.request,
				"",
			)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, invitation)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectID, invitation.ID)
			}
			fixture.orgRepo.AssertExpectations(t)
			fixture.invRepo.AssertExpectations(t)
			fixture.memberRepo.AssertExpectations(t)
			fixture.userSvc.AssertExpectations(t)
		})
	}
}

func hookErrorExecutor(t *testing.T, hookErr error) *ServiceHookExecutor {
	t.Helper()

	config := &types.OrganizationsServiceHooksConfig{Invitations: &types.OrganizationInvitationServiceHooks{}}
	config.Invitations.RegisterBeforeCreate(func(context.Context, *models.Actor, *types.OrganizationInvitation) error {
		return hookErr
	})
	return NewServiceHookExecutor(config, nil, nil)
}

func hookAfterCreateErrorExecutor(t *testing.T, hookErr error) *ServiceHookExecutor {
	t.Helper()

	config := &types.OrganizationsServiceHooksConfig{Invitations: &types.OrganizationInvitationServiceHooks{}}
	config.Invitations.RegisterAfterCreate(func(context.Context, *models.Actor, *types.OrganizationInvitation) error {
		return hookErr
	})
	return NewServiceHookExecutor(config, nil, nil)
}

func TestOrganizationInvitationService_SendOrganizationInvitationEmail(t *testing.T) {
	t.Parallel()

	invitation := &types.OrganizationInvitation{ID: "inv-1", Email: "bob@example.com", OrganizationID: "org-1", Role: "member"}
	organization := &types.Organization{ID: "org-1", Name: "Acme"}

	t.Run("renders the template and sends the email", func(t *testing.T) {
		t.Parallel()

		fixture := newInvitationExtraFixture(t, nil, nil)
		calls := make(chan invitationEmailCall, 1)
		fixture.mailer.called = calls

		require.NoError(t, fixture.svc.sendOrganizationInvitationEmail(context.Background(), invitation, organization, "https://app.example.com/invite"))

		call := <-calls
		require.Equal(t, "bob@example.com", call.to)
		require.Contains(t, call.subject, "Acme")
		require.Contains(t, call.text, "member")
		require.Contains(t, call.text, "https://app.example.com/invite")
		require.Contains(t, call.html, "Acme")
	})

	t.Run("reports a template render failure", func(t *testing.T) {
		t.Parallel()

		fixture := newInvitationExtraFixture(t, nil, func(f *invitationExtraFixture) {
			f.templateMgr = emailtmpl.NewManager()
		})

		err := fixture.svc.sendOrganizationInvitationEmail(context.Background(), invitation, organization, "https://app.example.com/invite")
		require.Error(t, err)
		require.NotEmpty(t, fixture.logger.errors)
		require.Contains(t, fixture.logger.errors, "failed to render organization invitation template")
	})
}

func TestOrganizationInvitationService_BuildOrganizationInvitationURL(t *testing.T) {
	t.Parallel()

	invitation := &types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1"}

	tests := []struct {
		name        string
		redirectURL string
		wantParams  map[string]string
		wantBase    string
	}{
		{
			name:        "falls back to the configured base url",
			redirectURL: "",
			wantBase:    "https://example.com/auth",
			wantParams:  map[string]string{"organization_id": "org-1", "invitation_id": "inv-1"},
		},
		{
			name:        "keeps an existing redirect query",
			redirectURL: "https://app.example.com/welcome?lang=en",
			wantBase:    "https://app.example.com/welcome",
			wantParams:  map[string]string{"lang": "en", "organization_id": "org-1", "invitation_id": "inv-1"},
		},
		{
			name:        "an unparsable redirect url is returned unchanged",
			redirectURL: "://bad",
			wantBase:    "://bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newInvitationExtraFixture(t, nil, nil)
			got := fixture.svc.buildOrganizationInvitationURL(invitation, tt.redirectURL)
			if tt.wantParams == nil {
				require.Equal(t, tt.wantBase, got)
				return
			}

			parsed, err := url.Parse(got)
			require.NoError(t, err)
			require.Equal(t, tt.wantBase, parsed.Scheme+"://"+parsed.Host+parsed.Path)
			for key, value := range tt.wantParams {
				require.Equal(t, value, parsed.Query().Get(key))
			}
		})
	}
}
