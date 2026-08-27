package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

func newAuthorizeOrgAccessUseCases(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) *UseCases {
	return &UseCases{
		authorizer:    rootservices.NewDefaultAuthorizer(),
		serviceUtils:  orgservices.NewServiceUtils(orgRepo, memberRepo, nil, 0),
		accessControl: accessControl,
	}
}

func newGetInvitationUseCases(invSvc orgservices.OrganizationInvitationService, userSvc rootservices.UserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) *UseCases {
	return &UseCases{
		invitationService: invSvc,
		userService:       userSvc,
		serviceUtils:      orgservices.NewServiceUtils(orgRepo, memberRepo, nil, 0),
		authorizer:        rootservices.NewDefaultAuthorizer(),
		accessControl:     accessControl,
	}
}

func TestUseCases_GetOrganizationInvitation(t *testing.T) {
	t.Parallel()

	invitation := &types.OrganizationInvitation{
		ID:             "inv-1",
		Email:          "bob@example.com",
		OrganizationID: "org-a",
		Role:           "admin",
	}
	response := &types.GetOrganizationInvitationResponse{
		Invitation:   invitation,
		Organization: types.OrganizationSummary{ID: "org-a", OwnerID: "owner-1", Name: "Org A", Slug: "org-a"},
	}

	tests := []struct {
		name      string
		actor     *models.Actor
		orgID     string
		setup     func(*orgtests.MockOrganizationInvitationService, *internaltests.MockUserService, *orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.AccessControlServiceStub)
		expectErr error
	}{
		{
			name:  "invitee can read own invitation",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "bob@example.com"}, nil).Once()
			},
		},
		{
			name:  "member without read permission is not found",
			actor: &models.Actor{ID: "user-2", Type: models.ActorUser},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "carol@example.com"}, nil).Once()
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-a", UserID: "user-2", Role: "viewer"}, nil).Once()
				accessControl.RolePermissions = map[string][]string{
					"viewer": {"organizations:members:read"},
				}
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:  "member with read permission can read",
			actor: &models.Actor{ID: "user-2", Type: models.ActorUser},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "carol@example.com"}, nil).Once()
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-a", UserID: "user-2", Role: "admin"}, nil).Once()
				accessControl.RolePermissions = map[string][]string{
					"admin": {"organizations:invitations:read"},
				}
			},
		},
		{
			name:  "non-member with global organizations:* scope is not found",
			actor: &models.Actor{ID: "user-3", Type: models.ActorUser, Scopes: []string{"organizations:*"}},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-3").Return(&models.User{ID: "user-3", Email: "eve@example.com"}, nil).Once()
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-3").Return((*types.OrganizationMember)(nil), nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:  "machine bound to org with scope can read",
			actor: &models.Actor{ID: "key-1", Type: models.ActorMachine, Scopes: []string{"organizations:invitations:read"}, Claims: map[string]any{"organization_id": "org-a"}},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "key-1").Return((*models.User)(nil), nil).Once()
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "owner-1"}, nil).Once()
			},
		},
		{
			name:  "machine bound to another org is not found",
			actor: &models.Actor{ID: "key-1", Type: models.ActorMachine, Scopes: []string{"organizations:invitations:read"}, Claims: map[string]any{"organization_id": "org-b"}},
			orgID: "org-a",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
				userSvc.On("GetByID", mock.Anything, "key-1").Return((*models.User)(nil), nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:  "invitation from another organization is not found",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID: "org-b",
			setup: func(invSvc *orgtests.MockOrganizationInvitationService, userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			invSvc := &orgtests.MockOrganizationInvitationService{}
			userSvc := &internaltests.MockUserService{}
			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			accessControl := orgtests.NewAccessControlServiceStub()
			if tt.setup != nil {
				tt.setup(invSvc, userSvc, orgRepo, memberRepo, accessControl)
			}

			uc := newGetInvitationUseCases(invSvc, userSvc, orgRepo, memberRepo, accessControl)
			resp, err := uc.GetOrganizationInvitation(context.Background(), tt.actor, tt.orgID, "inv-1")
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
			invSvc.AssertExpectations(t)
			userSvc.AssertExpectations(t)
			orgRepo.AssertExpectations(t)
			memberRepo.AssertExpectations(t)
		})
	}
}

func TestUseCases_authorizeOrgAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		actor         *models.Actor
		orgID         string
		requiredScope string
		setup         func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.AccessControlServiceStub)
		expectErr     error
	}{
		{
			name:          "admin in org A cannot delete org B when viewer there",
			actor:         &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:         "org-b",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-b").Return(&types.Organization{ID: "org-b", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-b", "user-1").Return(&types.OrganizationMember{ID: "mem-b", OrganizationID: "org-b", UserID: "user-1", Role: "viewer"}, nil).Once()
				accessControl.RolePermissions = map[string][]string{
					"viewer": {"organizations:members:read"},
					"admin":  {"organizations:delete"},
				}
			},
			expectErr: coreerrors.ErrInsufficientPermissions,
		},
		{
			name:          "admin role grants delete in org A",
			actor:         &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-1").Return(&types.OrganizationMember{ID: "mem-a", OrganizationID: "org-a", UserID: "user-1", Role: "admin"}, nil).Once()
				accessControl.RolePermissions = map[string][]string{
					"admin": {"organizations:delete"},
				}
			},
		},
		{
			name:          "wildcard role permission grants scope",
			actor:         &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-1").Return(&types.OrganizationMember{ID: "mem-a", OrganizationID: "org-a", UserID: "user-1", Role: "admin"}, nil).Once()
				accessControl.RolePermissions = map[string][]string{
					"admin": {"organizations:*"},
				}
			},
		},
		{
			name:          "user with no member row is forbidden",
			actor:         &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-1").Return((*types.OrganizationMember)(nil), nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:          "machine bound to org A is blocked on org B",
			actor:         &models.Actor{ID: "key-1", Type: models.ActorMachine, Scopes: []string{"organizations:delete"}, Claims: map[string]any{"organization_id": "org-a"}},
			orgID:         "org-b",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			expectErr:     coreerrors.ErrForbidden,
		},
		{
			name:          "machine bound to org A with matching scope is allowed",
			actor:         &models.Actor{ID: "key-1", Type: models.ActorMachine, Scopes: []string{"organizations:delete"}, Claims: map[string]any{"organization_id": "org-a"}},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
			},
		},
		{
			name:          "machine bound to org A without scope is denied",
			actor:         &models.Actor{ID: "key-1", Type: models.ActorMachine, Scopes: []string{"organizations:read"}, Claims: map[string]any{"organization_id": "org-a"}},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrInsufficientPermissions,
		},
		{
			name:          "dangling member role fails closed",
			actor:         &models.Actor{ID: "user-1", Type: models.ActorUser},
			orgID:         "org-a",
			requiredScope: orgconstants.OrganizationsDeletePermission,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, accessControl *orgtests.AccessControlServiceStub) {
				orgRepo.On("GetByID", mock.Anything, "org-a").Return(&types.Organization{ID: "org-a", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-a", "user-1").Return(&types.OrganizationMember{ID: "mem-a", OrganizationID: "org-a", UserID: "user-1", Role: "ghost"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			accessControl := orgtests.NewAccessControlServiceStub()
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo, accessControl)
			}

			uc := newAuthorizeOrgAccessUseCases(orgRepo, memberRepo, accessControl)
			err := uc.authorizeOrgAccess(context.Background(), tt.actor, tt.orgID, tt.requiredScope)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}
			orgRepo.AssertExpectations(t)
			memberRepo.AssertExpectations(t)
		})
	}
}
