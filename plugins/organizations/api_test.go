package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestAPI_GetUserPermissionsInOrganization(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		userID         string
		organizationID string
		setup          func(*orgtests.MockOrganizationService, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectPerms    []string
	}{
		{
			name:           "unauthorized without identifiers",
			userID:         "",
			organizationID: "",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "missing organization is forbidden",
			userID:         "user-1",
			organizationID: "org-1",
			setup: func(orgService *orgtests.MockOrganizationService, _ *orgtests.MockOrganizationMemberRepository) {
				orgService.On("ExistsByID", mock.Anything, "org-1").Return(false, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "existence check error is propagated",
			userID:         "user-1",
			organizationID: "org-1",
			setup: func(orgService *orgtests.MockOrganizationService, _ *orgtests.MockOrganizationMemberRepository) {
				orgService.On("ExistsByID", mock.Anything, "org-1").Return(false, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "non-member is forbidden",
			userID:         "user-1",
			organizationID: "org-1",
			setup: func(orgService *orgtests.MockOrganizationService, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgService.On("ExistsByID", mock.Anything, "org-1").Return(true, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "member receives role permissions",
			userID:         "user-1",
			organizationID: "org-1",
			setup: func(orgService *orgtests.MockOrganizationService, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgService.On("ExistsByID", mock.Anything, "org-1").Return(true, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
			},
			expectPerms: []string{"organizations:read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgService := &orgtests.MockOrganizationService{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgService, memberRepo)
			}

			accessControl := orgtests.NewAccessControlServiceStub()
			accessControl.RolePermissions["member"] = []string{"organizations:read"}

			api := &API{
				organizationService:  orgService,
				memberRepo:           memberRepo,
				accessControlService: accessControl,
			}

			perms, err := api.GetUserPermissionsInOrganization(context.Background(), tt.userID, tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectPerms, perms)
			orgService.AssertExpectations(t)
			memberRepo.AssertExpectations(t)
		})
	}
}
