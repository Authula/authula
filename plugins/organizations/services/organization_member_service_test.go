package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

func newTestOrganizationMemberService(userSvc *internaltests.MockUserService, accessControlService rootservices.AccessControlService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, membersLimit *int) *OrganizationMemberService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	return NewOrganizationMemberService(userSvc, accessControlService, orgRepo, memberRepo, membersLimit, &orgtests.MockTxRunner{}, serviceUtils)
}

func TestOrganizationMemberService_AddMember(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")
	zeroLimit := 0
	threeLimit := 3
	twoLimit := 2

	tests := []struct {
		name                 string
		actorUserID          string
		organizationID       string
		request              types.AddOrganizationMemberRequest
		accessControlService rootservices.AccessControlService
		membersLimit         *int
		setup                func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *internaltests.MockUserService, *orgtests.MockOrganizationMemberHooks)
		expectErr            error
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "forbidden for non owner",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "bad request empty user id",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrUnprocessableEntity,
		},
		{
			name:           "bad request empty role",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: ""},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrUnprocessableEntity,
		},
		{
			name:           "invalid role is rejected",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "ghost"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrBadRequest,
		},
		{
			name:           "zero limit treated as unlimited",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			membersLimit:   &zeroLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2" && member.Role == "member"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
		},
		{
			name:           "success within limit",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			membersLimit:   &threeLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2" && member.Role == "member"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
		},
		{
			name:           "quota exceeded",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			membersLimit:   &twoLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(2, nil).Once()
			},
			expectErr: orgconstants.ErrMembersQuotaExceeded,
		},
		{
			name:           "count lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			membersLimit:   &twoLimit,
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("CountByOrganizationID", mock.Anything, "org-1").Return(0, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:                 "higher role is forbidden",
			actorUserID:          "user-2",
			organizationID:       "org-1",
			request:              types.AddOrganizationMemberRequest{UserID: "user-3", Role: "manager"},
			accessControlService: orgtests.NewAccessControlServiceStubWithWeights(nil, map[string]int{"user-2": 10}),
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-3").Return(&models.User{ID: "user-3", Email: "user3@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-3").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:                 "access control forbidden is normalized",
			actorUserID:          "user-2",
			organizationID:       "org-1",
			request:              types.AddOrganizationMemberRequest{UserID: "user-3", Role: "manager"},
			accessControlService: &orgtests.AccessControlServiceStub{Err: errors.New("forbidden")},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-3").Return(&models.User{ID: "user-3", Email: "user3@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-3").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "user lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "user not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "existing member conflict",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrConflict,
		},
		{
			name:           "lookup existing member error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "create error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2" && member.Role == "member"
				})).Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-2" && member.Role == "member"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			userSvc := &internaltests.MockUserService{}
			hooks := &orgtests.MockOrganizationMemberHooks{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo, userSvc, hooks)
			}

			accessControlService := tt.accessControlService
			if accessControlService == nil {
				accessControlService = orgtests.NewAccessControlServiceStub()
			}

			svc := newTestOrganizationMemberService(userSvc, accessControlService, orgRepo, memberRepo, tt.membersLimit)
			member, err := svc.AddMember(context.Background(), tt.actorUserID, tt.organizationID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				if strings.Contains(tt.name, "invalid role") {
					require.ErrorContains(t, err, tt.expectErr.Error())
				} else {
					require.ErrorIs(t, err, tt.expectErr)
				}
				if tt.setup != nil {
					require.True(t, orgRepo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
					require.True(t, userSvc.AssertExpectations(t))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, member)
			if tt.setup != nil {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, userSvc.AssertExpectations(t))
			}
		})
	}
}

func TestOrganizationMemberService_GetAllMembers(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectLen      int
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationID", mock.Anything, "org-1", 1, 10).Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationID", mock.Anything, "org-1", 1, 10).Return([]types.OrganizationMember{{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}}, nil).Once()
			},
			expectLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			userService := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo)
			}

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			members, err := svc.GetAllMembers(context.Background(), tt.actorUserID, tt.organizationID, 1, 10)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				if tt.setup != nil {
					require.True(t, orgRepo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, members, tt.expectLen)
			if tt.setup != nil {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
			}
		})
	}
}

func TestOrganizationMemberService_GetMember(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		memberID       string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectMemberID string
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			memberID:       "mem-1",
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "member id empty",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: internalerrors.ErrUnprocessableEntity,
		},
		{
			name:           "member id whitespace",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "missing",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "missing").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "not found when member is missing",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "not found when member belongs to another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectMemberID: "mem-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			userService := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo)
			}

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			member, err := svc.GetMember(context.Background(), tt.actorUserID, tt.organizationID, tt.memberID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				if tt.setup != nil {
					require.True(t, orgRepo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, member)
			require.Equal(t, tt.expectMemberID, member.ID)
			if tt.setup != nil {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
			}
		})
	}
}

func TestOrganizationMemberService_UpdateMember(t *testing.T) {
	t.Parallel()

	updateErr := errors.New("update error")
	repoErr := errors.New("repository error")

	tests := []struct {
		name                 string
		actorUserID          string
		organizationID       string
		memberID             string
		request              types.UpdateOrganizationMemberRequest
		accessControlService rootservices.AccessControlService
		setup                func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationMemberHooks)
		expectErr            error
		expectRole           string
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "repository error fetching member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "not found when member is missing",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "not found when member belongs to another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-2", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "bad request empty role",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: ""},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrBadRequest,
		},
		{
			name:           "invalid role is rejected",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "ghost"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrBadRequest,
		},
		{
			name:                 "higher role is forbidden",
			actorUserID:          "user-2",
			organizationID:       "org-1",
			memberID:             "mem-1",
			request:              types.UpdateOrganizationMemberRequest{Role: "manager"},
			accessControlService: orgtests.NewAccessControlServiceStubWithWeights(nil, map[string]int{"user-2": 10}),
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-3", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:                 "access control forbidden is normalized",
			actorUserID:          "user-2",
			organizationID:       "org-1",
			memberID:             "mem-1",
			request:              types.UpdateOrganizationMemberRequest{Role: "manager"},
			accessControlService: &orgtests.AccessControlServiceStub{Err: errors.New("forbidden")},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-3", Role: "member"}, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "update error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.ID == "mem-1" && member.Role == "admin"
				})).Return(nil, updateErr).Once()
			},
			expectErr: updateErr,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.ID == "mem-1" && member.Role == "admin"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}, nil).Once()
			},
			expectRole: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationMemberHooks{}
			userService := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo, hooks)
			}

			accessControlService := tt.accessControlService
			if accessControlService == nil {
				accessControlService = orgtests.NewAccessControlServiceStub()
			}

			svc := newTestOrganizationMemberService(userService, accessControlService, orgRepo, memberRepo, nil)
			member, err := svc.UpdateMember(context.Background(), tt.actorUserID, tt.organizationID, tt.memberID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				if strings.Contains(tt.name, "invalid role") {
					require.ErrorContains(t, err, tt.expectErr.Error())
				} else {
					require.ErrorIs(t, err, tt.expectErr)
				}
				if tt.setup != nil {
					require.True(t, orgRepo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, member)
			require.Equal(t, tt.expectRole, member.Role)
			if tt.setup != nil {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
			}
		})
	}
}

func TestOrganizationMemberService_RemoveMember(t *testing.T) {
	t.Parallel()

	deleteErr := errors.New("delete error")
	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		memberID       string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationMemberHooks)
		expectErr      error
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			memberID:       "mem-1",
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "repository error fetching member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "not found when member is missing",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "not found when member belongs to another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr: internalerrors.ErrNotFound,
		},
		{
			name:           "delete error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("Delete", mock.Anything, "mem-1").Return(deleteErr).Once()
			},
			expectErr: deleteErr,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("Delete", mock.Anything, "mem-1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationMemberHooks{}
			userService := &internaltests.MockUserService{}
			if tt.setup != nil {
				tt.setup(orgRepo, memberRepo, hooks)
			}

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			err := svc.RemoveMember(context.Background(), tt.actorUserID, tt.organizationID, tt.memberID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				if tt.setup != nil {
					require.True(t, orgRepo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
				}
				return
			}
			require.NoError(t, err)
			if tt.setup != nil {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
			}
		})
	}
}
