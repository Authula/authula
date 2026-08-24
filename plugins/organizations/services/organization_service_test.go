package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationService_CreateOrganization(t *testing.T) {
	t.Parallel()

	zeroLimit := 0
	threeLimit := 3
	twoLimit := 2
	accessControlService := orgtests.NewAccessControlServiceStub()
	txRunner := &orgtests.MockTxRunner{}

	successSetup := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
		repo.On("Create", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
			return org != nil && org.OwnerID == "user-1" && org.Name == "Acme Inc" && org.Slug == "acme-inc" && string(internaltests.MarshalToJSON(t, org.Metadata)) == `{"tier":"pro"}`
		})).Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "pro"}}, nil).Once()
		memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
			return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-1" && member.Role == "member"
		})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
	}

	limitSuccessSetup := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
		repo.On("CountAccessibleByUserID", mock.Anything, "user-1").Return(2, nil).Once()
		repo.On("Create", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
			return org != nil && org.OwnerID == "user-1" && org.Name == "Acme Labs" && org.Slug == "acme-labs" && string(internaltests.MarshalToJSON(t, org.Metadata)) == "{}"
		})).Return(&types.Organization{ID: "org-2", OwnerID: "user-1", Name: "Acme Labs", Slug: "acme-labs", Metadata: map[string]any{}}, nil).Once()
		memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
			return member != nil && member.OrganizationID == "org-2" && member.UserID == "user-1" && member.Role == "member"
		})).Return(&types.OrganizationMember{ID: "mem-3", OrganizationID: "org-2", UserID: "user-1", Role: "member"}, nil).Once()
	}

	quotaExceededSetup := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
		repo.On("CountAccessibleByUserID", mock.Anything, "user-1").Return(2, nil).Once()
	}

	tests := []struct {
		name              string
		actorUserID       string
		organizationLimit *int
		request           types.CreateOrganizationRequest
		setup             func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationHooks, *ServiceUtils)
		expectErr         error
		expectCalled      bool
		expectReturned    string
	}{
		{
			name:        "unauthorized",
			actorUserID: "",
			request:     types.CreateOrganizationRequest{Name: "Acme", Role: "member"},
			expectErr:   coreerrors.ErrUnauthorized,
		},
		{
			name:        "missing role",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "Acme"},
			expectErr:   coreerrors.ErrUnprocessableEntity,
		},
		{
			name:        "bad request",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "", Role: "member"},
			expectErr:   coreerrors.ErrUnprocessableEntity,
		},
		{
			name:        "invalid role",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "Acme", Role: "ghost"},
			expectErr:   coreerrors.ErrUnprocessableEntity,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			request:        types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member", Metadata: map[string]any{"tier": "pro"}},
			setup:          successSetup,
			expectCalled:   true,
			expectReturned: "org-1",
		},
		{
			name:              "zero limit treated as unlimited",
			actorUserID:       "user-1",
			organizationLimit: &zeroLimit,
			request:           types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member", Metadata: map[string]any{"tier": "pro"}},
			setup:             successSetup,
			expectCalled:      true,
			expectReturned:    "org-1",
		},
		{
			name:              "success within limit",
			actorUserID:       "user-1",
			organizationLimit: &threeLimit,
			request:           types.CreateOrganizationRequest{Name: "Acme Labs", Role: "member"},
			setup:             limitSuccessSetup,
			expectCalled:      true,
			expectReturned:    "org-2",
		},
		{
			name:              "quota exceeded across owned and member organizations",
			actorUserID:       "user-1",
			organizationLimit: &twoLimit,
			request:           types.CreateOrganizationRequest{Name: "Acme Platform", Role: "member"},
			setup:             quotaExceededSetup,
			expectErr:         constants.ErrOrganizationsQuotaExceeded,
			expectCalled:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			hooks := &orgtests.MockOrganizationHooks{}
			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			if tt.setup != nil {
				tt.setup(repo, memberRepo, hooks, serviceUtils)
			}

			svc := NewOrganizationService(repo, memberRepo, serviceUtils, accessControlService, tt.organizationLimit, txRunner)
			org, err := svc.CreateOrganization(context.Background(), orgtests.Actor(tt.actorUserID), tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				if tt.expectCalled {
					require.True(t, repo.AssertExpectations(t))
					require.True(t, memberRepo.AssertExpectations(t))
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, org)
			if tt.expectReturned != "" {
				require.Equal(t, tt.expectReturned, org.ID)
			}
			if tt.expectCalled {
				repo.AssertExpectations(t)
				require.True(t, memberRepo.AssertExpectations(t))
			}
		})
	}
}

func TestOrganizationService_CreateOrganizationRequiresPrivilegedRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		role          string
		rolePerms     map[string][]string
		expectErr     error
		expectCreated bool
	}{
		{
			name:          "role with organizations:* wildcard is accepted",
			role:          "admin",
			rolePerms:     map[string][]string{"admin": {"organizations:*"}},
			expectCreated: true,
		},
		{
			name:          "role with all organization permissions enumerated is accepted",
			role:          "admin",
			rolePerms:     map[string][]string{"admin": constants.OrganizationPermissions},
			expectCreated: true,
		},
		{
			name:          "universal wildcard role is accepted",
			role:          "admin",
			rolePerms:     map[string][]string{"admin": {"*"}},
			expectCreated: true,
		},
		{
			name:      "role without all organization permissions is rejected",
			role:      "member",
			rolePerms: map[string][]string{"member": {"organizations:members:read"}},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			accessControl := orgtests.NewAccessControlServiceStub()
			accessControl.RolePermissions = tt.rolePerms

			repo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			svc := NewOrganizationService(repo, memberRepo, serviceUtils, accessControl, nil, &orgtests.MockTxRunner{})

			if tt.expectCreated {
				repo.On("Create", mock.Anything, mock.Anything).Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"}, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.Anything).Return(&types.OrganizationMember{ID: "mem-1"}, nil).Once()
			}

			org, err := svc.CreateOrganization(context.Background(), orgtests.Actor("user-1"), types.CreateOrganizationRequest{Name: "Acme", Role: tt.role})
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, org)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, org)
			repo.AssertExpectations(t)
			memberRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationService_GetAllOrganizations(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name             string
		actorUserID      string
		params           pagination.Params
		setup            func(*orgtests.MockOrganizationRepository)
		expectErr        error
		expectLen        int
		expectPagination pagination.Pagination
	}{
		{
			name:        "unauthorized",
			actorUserID: "",
			params:      pagination.Params{Page: 1, Limit: 10},
			expectErr:   coreerrors.ErrUnauthorized,
		},
		{
			name:        "returns the requested page with its metadata",
			actorUserID: "user-1",
			params:      pagination.Params{Page: 1, Limit: 10},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("GetAllAccessibleByUserID", mock.Anything, "user-1", 1, 10).
					Return([]types.Organization{
						{ID: "org-1", OwnerID: "user-1", Name: "Acme"},
						{ID: "org-2", OwnerID: "owner-2", Name: "Platform"},
					}, 2, nil).Once()
			},
			expectLen:        2,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: 2, TotalPages: 1, HasMore: false},
		},
		{
			name:        "out of range params are clamped before reaching the repository",
			actorUserID: "user-1",
			params:      pagination.Params{Page: -4, Limit: 5000},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("GetAllAccessibleByUserID", mock.Anything, "user-1", 1, pagination.MaxLimit).
					Return([]types.Organization{}, 0, nil).Once()
			},
			expectLen:        0,
			expectPagination: pagination.Pagination{Page: 1, Limit: pagination.MaxLimit, Total: 0, TotalPages: 0, HasMore: false},
		},
		{
			name:        "nil result is normalised to an empty slice",
			actorUserID: "user-1",
			params:      pagination.Params{Page: 1, Limit: 10},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("GetAllAccessibleByUserID", mock.Anything, "user-1", 1, 10).
					Return(([]types.Organization)(nil), 0, nil).Once()
			},
			expectLen:        0,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: 0, TotalPages: 0, HasMore: false},
		},
		{
			name:        "repository error is propagated",
			actorUserID: "user-1",
			params:      pagination.Params{Page: 1, Limit: 10},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("GetAllAccessibleByUserID", mock.Anything, "user-1", 1, 10).
					Return(([]types.Organization)(nil), 0, repoErr).Once()
			},
			expectErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(repo)
			}

			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			svc := NewOrganizationService(repo, memberRepo, serviceUtils, nil, nil, nil)
			resp, err := svc.GetAllOrganizations(context.Background(), orgtests.Actor(tt.actorUserID), tt.params)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, repo.AssertExpectations(t))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			require.Len(t, resp.Data, tt.expectLen)
			require.Equal(t, tt.expectPagination, resp.Pagination)
			require.True(t, repo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

// The organization quota must be enforced from a SQL count: the previous
// implementation counted a fetched list, which a page size would silently cap.
func TestOrganizationService_EnsureOrganizationLimit(t *testing.T) {
	t.Parallel()

	countErr := errors.New("count error")
	limit := 3

	tests := []struct {
		name              string
		actorUserID       string
		organizationLimit *int
		setup             func(*orgtests.MockOrganizationRepository)
		expectErr         error
		expectNoCount     bool
	}{
		{
			name:              "unauthorized actor is rejected before counting",
			actorUserID:       "",
			organizationLimit: &limit,
			expectErr:         coreerrors.ErrUnauthorized,
			expectNoCount:     true,
		},
		{
			name:          "no configured limit skips the count entirely",
			actorUserID:   "user-1",
			expectNoCount: true,
		},
		{
			name:              "count one below the limit is allowed",
			actorUserID:       "user-1",
			organizationLimit: &limit,
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("CountAccessibleByUserID", mock.Anything, "user-1").Return(limit-1, nil).Once()
			},
		},
		{
			name:              "count at the limit is rejected",
			actorUserID:       "user-1",
			organizationLimit: &limit,
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("CountAccessibleByUserID", mock.Anything, "user-1").Return(limit, nil).Once()
			},
			expectErr: constants.ErrOrganizationsQuotaExceeded,
		},
		{
			name:              "count error is propagated",
			actorUserID:       "user-1",
			organizationLimit: &limit,
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("CountAccessibleByUserID", mock.Anything, "user-1").Return(0, countErr).Once()
			},
			expectErr: countErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(repo)
			}

			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			svc := NewOrganizationService(repo, memberRepo, serviceUtils, nil, tt.organizationLimit, nil)

			err := svc.ensureOrganizationLimit(context.Background(), orgtests.Actor(tt.actorUserID), repo)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}

			if tt.expectNoCount {
				repo.AssertNotCalled(t, "CountAccessibleByUserID", mock.Anything, mock.Anything)
			}
			require.True(t, repo.AssertExpectations(t))
		})
	}
}

func TestOrganizationService_GetOrganizationByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
	}{
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "success for member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
			},
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(repo, memberRepo)
			}

			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			svc := NewOrganizationService(repo, memberRepo, serviceUtils, nil, nil, nil)
			org, err := svc.GetOrganizationByID(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, org)
		})
	}
}

func TestOrganizationService_UpdateOrganization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		request        types.UpdateOrganizationRequest
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationHooks, *ServiceUtils)
		expectErr      error
	}{
		{
			name:           "unauthorized if not user ID provided",
			actorUserID:    "",
			organizationID: "",
			request:        types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "unauthorized if no organization ID provided",
			actorUserID:    "user-1",
			organizationID: "",
			request:        types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "forbidden for non-owner member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1", Name: "Acme", Slug: "acme"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.ID == "org-1" && org.Name == "Acme Platform" && org.Slug == "acme"
				})).Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Platform", Slug: "acme"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			hooks := &orgtests.MockOrganizationHooks{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			if tt.setup != nil {
				tt.setup(repo, memberRepo, hooks, serviceUtils)
			}

			svc := NewOrganizationService(repo, memberRepo, serviceUtils, nil, nil, nil)
			org, err := svc.UpdateOrganization(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, org)
		})
	}
}

func TestOrganizationService_DeleteOrganization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationHooks, *ServiceUtils)
		expectErr      error
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(repo *orgtests.MockOrganizationRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				repo.On("Delete", mock.Anything, "org-1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			hooks := &orgtests.MockOrganizationHooks{}
			serviceUtils := &ServiceUtils{orgRepo: repo}
			if tt.setup != nil {
				tt.setup(repo, hooks, serviceUtils)
			}

			svc := NewOrganizationService(repo, nil, serviceUtils, nil, nil, nil)
			err := svc.DeleteOrganization(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
