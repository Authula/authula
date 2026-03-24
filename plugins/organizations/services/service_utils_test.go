package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestServiceUtils_authorizeOwner(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("organization repo failed")

	tests := []struct {
		name         string
		actorUserID  string
		organization string
		setup        func(*orgtests.MockOrganizationRepository)
		expectErr    error
		expectOwner  string
	}{
		{
			name:         "success",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectOwner: "user-1",
		},
		{
			name:         "unauthorized when inputs are missing",
			actorUserID:  "",
			organization: "org-1",
			expectErr:    internalerrors.ErrUnauthorized,
		},
		{
			name:         "not found when organization is missing",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "forbidden when actor is not owner",
			actorUserID:  "user-2",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:         "propagates repository error",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo)
			}

			org, err := (&ServiceUtils{orgRepo: orgRepo}).authorizeOwner(context.Background(), tt.actorUserID, tt.organization)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, org)
				orgRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, org)
			require.Equal(t, tt.expectOwner, org.OwnerID)
			orgRepo.AssertExpectations(t)
		})
	}
}

func TestServiceUtils_authorizeOrganizationAccess(t *testing.T) {
	t.Parallel()

	orgErr := errors.New("organization repo failed")
	memberErr := errors.New("member repo failed")

	tests := []struct {
		name         string
		actorUserID  string
		organization string
		setup        func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr    error
		expectOwner  string
		expectMember string
	}{
		{
			name:         "owner access",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectOwner: "user-1",
		},
		{
			name:         "member access",
			actorUserID:  "user-2",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2"}, nil).Once()
			},
			expectMember: "user-2",
		},
		{
			name:         "unauthorized when inputs are missing",
			actorUserID:  "",
			organization: "org-1",
			expectErr:    internalerrors.ErrUnauthorized,
		},
		{
			name:         "not found when organization is missing",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:         "forbidden when member is missing",
			actorUserID:  "user-2",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return((*types.OrganizationMember)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:         "propagates organization repository error",
			actorUserID:  "user-1",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), orgErr).Once()
			},
			expectErr: orgErr,
		},
		{
			name:         "propagates member repository error",
			actorUserID:  "user-2",
			organization: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return((*types.OrganizationMember)(nil), memberErr).Once()
			},
			expectErr: memberErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo)
			}

			org, member, err := (&ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}).authorizeOrganizationAccess(context.Background(), tt.actorUserID, tt.organization)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, org)
				require.Nil(t, member)
				orgRepo.AssertExpectations(t)
				memberRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, org)
			if tt.expectMember != "" {
				require.NotNil(t, member)
				require.Equal(t, tt.expectMember, member.UserID)
			} else {
				require.Nil(t, member)
			}
			orgRepo.AssertExpectations(t)
			memberRepo.AssertExpectations(t)
		})
	}
}

func TestServiceUtils_ensureOrganizationMembersLimit(t *testing.T) {
	t.Parallel()

	countErr := errors.New("count failed")
	zero := 0
	one := 1
	two := 2

	tests := []struct {
		name      string
		limit     *int
		setup     func(*orgtests.MockOrganizationMemberRepository)
		expectErr error
	}{
		{
			name:  "nil limit bypasses check",
			limit: nil,
		},
		{
			name:  "zero limit bypasses check",
			limit: &zero,
		},
		{
			name:  "under limit succeeds",
			limit: &two,
			setup: func(memberRepo *orgtests.MockOrganizationMemberRepository) {
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(1, nil).Once()
			},
		},
		{
			name:  "at limit returns quota exceeded",
			limit: &one,
			setup: func(memberRepo *orgtests.MockOrganizationMemberRepository) {
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(1, nil).Once()
			},
			expectErr: orgconstants.ErrMembersQuotaExceeded,
		},
		{
			name:  "count error propagates",
			limit: &two,
			setup: func(memberRepo *orgtests.MockOrganizationMemberRepository) {
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(0, countErr).Once()
			},
			expectErr: countErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(memberRepo)
			}

			err := (&ServiceUtils{}).ensureOrganizationMembersLimit(context.Background(), memberRepo, "org-1", tt.limit)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				memberRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			memberRepo.AssertExpectations(t)
		})
	}
}

func TestServiceUtils_ensureOrganizationInvitationsLimit(t *testing.T) {
	t.Parallel()

	countErr := errors.New("count failed")
	zero := 0
	one := 1
	two := 2

	tests := []struct {
		name      string
		limit     *int
		setup     func(*orgtests.MockOrganizationInvitationRepository)
		expectErr error
	}{
		{
			name:  "nil limit bypasses check",
			limit: nil,
		},
		{
			name:  "zero limit bypasses check",
			limit: &zero,
		},
		{
			name:  "under limit succeeds",
			limit: &two,
			setup: func(invRepo *orgtests.MockOrganizationInvitationRepository) {
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(1, nil).Once()
			},
		},
		{
			name:  "at limit returns quota exceeded",
			limit: &one,
			setup: func(invRepo *orgtests.MockOrganizationInvitationRepository) {
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(1, nil).Once()
			},
			expectErr: orgconstants.ErrInvitationsQuotaExceeded,
		},
		{
			name:  "count error propagates",
			limit: &two,
			setup: func(invRepo *orgtests.MockOrganizationInvitationRepository) {
				invRepo.On("CountByOrganizationIDAndEmail", mock.Anything, "org-1", "user@example.com").Return(0, countErr).Once()
			},
			expectErr: countErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			invRepo := &orgtests.MockOrganizationInvitationRepository{}
			if tt.setup != nil {
				tt.setup(invRepo)
			}

			err := (&ServiceUtils{}).ensureOrganizationInvitationsLimit(context.Background(), invRepo, "org-1", "user@example.com", tt.limit)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				invRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			invRepo.AssertExpectations(t)
		})
	}
}

func TestServiceUtils_ensureEmailVerifiedForInvitationAcceptance(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("user lookup failed")

	tests := []struct {
		name                 string
		userID               string
		requireEmailVerified bool
		setup                func(*internaltests.MockUserService)
		expectErr            error
		expectUser           *models.User
	}{
		{
			name:                 "returns user when verification is disabled",
			userID:               "user-1",
			requireEmailVerified: false,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false}, nil).Once()
			},
			expectUser: &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false},
		},
		{
			name:                 "returns user when verification is required and user is verified",
			userID:               "user-1",
			requireEmailVerified: true,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true}, nil).Once()
			},
			expectUser: &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: true},
		},
		{
			name:                 "forbids unverified user when verification is required",
			userID:               "user-1",
			requireEmailVerified: true,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:                 "returns not found when user is missing",
			userID:               "user-1",
			requireEmailVerified: true,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:                 "returns not found when user has no email",
			userID:               "user-1",
			requireEmailVerified: true,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", EmailVerified: true}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:                 "returns not found when user ID is empty",
			userID:               "",
			requireEmailVerified: true,
			expectErr:            internalerrors.ErrNotFound,
		},
		{
			name:                 "propagates user service error",
			userID:               "user-1",
			requireEmailVerified: true,
			setup: func(userSvc *internaltests.MockUserService) {
				userSvc.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), lookupErr).Once()
			},
			expectErr: lookupErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userSvc := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(userSvc)
			}

			user, err := (&ServiceUtils{}).ensureEmailVerifiedForInvitationAcceptance(context.Background(), userSvc, tt.userID, tt.requireEmailVerified)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, user)
				userSvc.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)
			require.Equal(t, tt.expectUser.ID, user.ID)
			require.Equal(t, tt.expectUser.Email, user.Email)
			require.Equal(t, tt.expectUser.EmailVerified, user.EmailVerified)
			userSvc.AssertExpectations(t)
		})
	}
}

func TestServiceUtils_slugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "lowercases text", input: "Hello World", expect: "hello-world"},
		{name: "collapses separators", input: "Hello---World__Again", expect: "hello-world-again"},
		{name: "trims edge separators", input: "--Hello World--", expect: "hello-world"},
		{name: "keeps alphanumeric characters", input: "Team 42A", expect: "team-42a"},
		{name: "returns empty for punctuation only", input: "!!!", expect: ""},
	}

	utils := &ServiceUtils{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expect, utils.slugify(tt.input))
		})
	}
}
