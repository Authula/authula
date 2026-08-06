package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
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
		serviceUtils:  orgservices.NewServiceUtils(orgRepo, memberRepo, nil, nil),
		accessControl: accessControl,
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
