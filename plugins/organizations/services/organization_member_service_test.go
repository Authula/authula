package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

func newTestOrganizationMemberService(userSvc *internaltests.MockUserService, accessControlService rootservices.AccessControlService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, membersLimit *int) *organizationMemberService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	return NewOrganizationMemberService(userSvc, accessControlService, orgRepo, memberRepo, membersLimit, &orgtests.MockTxRunner{}, serviceUtils)
}

func expectActorMember(memberRepo *orgtests.MockOrganizationMemberRepository, organizationID, actorUserID string) {
	if actorUserID == "" || organizationID == "" {
		return
	}
	memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, organizationID, actorUserID).Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: organizationID, UserID: actorUserID, Role: "admin"}, nil).Maybe()
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
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "bad request empty user id",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "", Role: "member"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:           "bad request empty role",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.AddOrganizationMemberRequest{UserID: "user-2", Role: ""},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
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
			expectErr: coreerrors.ErrBadRequest,
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
			expectErr: coreerrors.ErrForbidden,
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
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrConflict,
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			accessControlService := tt.accessControlService
			if accessControlService == nil {
				accessControlService = orgtests.NewAccessControlServiceStub()
			}

			svc := newTestOrganizationMemberService(userSvc, accessControlService, orgRepo, memberRepo, tt.membersLimit)
			member, err := svc.AddMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.request)
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
		name             string
		actorUserID      string
		organizationID   string
		params           pagination.Params
		setup            func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr        error
		expectLen        int
		expectPagination pagination.Pagination
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationIDWithUser", mock.Anything, "org-1", 1, 10).Return(nil, 0, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationIDWithUser", mock.Anything, "org-1", 1, 10).
					Return([]types.OrganizationMemberResponse{{ID: "mem-1", OrganizationID: "org-1", Role: "member"}}, 25, nil).Once()
			},
			expectLen:        1,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: 25, TotalPages: 3, HasMore: true},
		},
		{
			name:           "a negative page is clamped before reaching the repository",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: -4, Limit: 5000},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationIDWithUser", mock.Anything, "org-1", 1, 5000).
					Return([]types.OrganizationMemberResponse{}, 0, nil).Once()
			},
			expectLen:        0,
			expectPagination: pagination.Pagination{Page: 1, Limit: 5000, Total: 0, TotalPages: 0, HasMore: false},
		},
		{
			name:           "nil result is normalised to an empty slice",
			actorUserID:    "user-1",
			organizationID: "org-1",
			params:         pagination.Params{Page: 1, Limit: 10},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetAllByOrganizationIDWithUser", mock.Anything, "org-1", 1, 10).
					Return(([]types.OrganizationMemberResponse)(nil), 0, nil).Once()
			},
			expectLen:        0,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: 0, TotalPages: 0, HasMore: false},
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			resp, err := svc.GetAllMembers(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.params)
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
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			require.Len(t, resp.Data, tt.expectLen)
			require.Equal(t, tt.expectPagination, resp.Pagination)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
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
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "member id empty",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:           "member id whitespace",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "missing",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByIDWithUser", mock.Anything, "missing").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByIDWithUser", mock.Anything, "mem-1").Return(nil, repoErr).Once()
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
				memberRepo.On("GetByIDWithUser", mock.Anything, "mem-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "not found when member belongs to another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByIDWithUser", mock.Anything, "mem-1").Return(&types.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByIDWithUser", mock.Anything, "mem-1").Return(&types.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			member, err := svc.GetMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.memberID)
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

func TestOrganizationMemberService_GetMemberByUserID(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		userID         string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectMemberID string
	}{
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			userID:         "user-2",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "empty user id",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserIDWithUser", mock.Anything, "org-1", "user-2").Return(nil, repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "not found when member does not exist",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserIDWithUser", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserIDWithUser", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			member, err := svc.GetMemberByUserID(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.userID)
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
			expectErr:      coreerrors.ErrUnauthorized,
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
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrForbidden,
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
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrBadRequest,
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
			expectErr: coreerrors.ErrBadRequest,
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
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "non-owner cannot change the owner's role",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-owner",
			request:        types.UpdateOrganizationMemberRequest{Role: "viewer"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-owner").Return(&types.OrganizationMember{ID: "mem-owner", OrganizationID: "org-1", UserID: "owner-1", Role: "owner"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "owner can change another member's role",
			actorUserID:    "owner-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			request:        types.UpdateOrganizationMemberRequest{Role: "admin"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("Update", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
					return member != nil && member.ID == "mem-1" && member.Role == "admin"
				})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}, nil).Once()
			},
			expectRole: "admin",
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			accessControlService := tt.accessControlService
			if accessControlService == nil {
				accessControlService = orgtests.NewAccessControlServiceStub()
			}

			svc := newTestOrganizationMemberService(userService, accessControlService, orgRepo, memberRepo, nil)
			member, err := svc.UpdateMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.memberID, tt.request)
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
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrForbidden,
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
			expectErr: coreerrors.ErrNotFound,
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
			expectErr: coreerrors.ErrNotFound,
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
			name:           "owner cannot remove their own membership",
			actorUserID:    "user-1",
			organizationID: "org-1",
			memberID:       "mem-owner",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationMemberHooks) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByID", mock.Anything, "mem-owner").Return(&types.OrganizationMember{ID: "mem-owner", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
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
			expectActorMember(memberRepo, tt.organizationID, tt.actorUserID)

			svc := newTestOrganizationMemberService(userService, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
			err := svc.RemoveMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.memberID)
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

func TestOrganizationMemberService_RemoveMemberGuards(t *testing.T) {
	t.Parallel()

	t.Run("cannot remove member with heavier role", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}, nil).Once()

		svc := newTestOrganizationMemberService(&internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
		err := svc.RemoveMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1")
		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		orgRepo.AssertExpectations(t)
		memberRepo.AssertExpectations(t)
	})

	t.Run("cannot remove owner unless actor is the owner", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}, nil).Once()
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "admin"}, nil).Once()

		svc := newTestOrganizationMemberService(&internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
		err := svc.RemoveMember(context.Background(), orgtests.Actor("user-2"), "org-1", "mem-1")
		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		orgRepo.AssertExpectations(t)
		memberRepo.AssertExpectations(t)
	})

	t.Run("owner can remove a member", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-actor", OrganizationID: "org-1", UserID: "user-1", Role: "admin"}, nil).Once()
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
		memberRepo.On("Delete", mock.Anything, "mem-1").Return(nil).Once()

		svc := newTestOrganizationMemberService(&internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
		err := svc.RemoveMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1")
		require.NoError(t, err)
		orgRepo.AssertExpectations(t)
		memberRepo.AssertExpectations(t)
	})
}
