package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
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
			return org != nil && org.OwnerID == "user-1" && org.Name == "Acme Inc" && org.Slug == "acme-inc" && string(org.Metadata) == `{"tier":"pro"}`
		})).Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: json.RawMessage(`{"tier":"pro"}`)}, nil).Once()
		memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
			return member != nil && member.OrganizationID == "org-1" && member.UserID == "user-1" && member.Role == "member"
		})).Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
	}

	limitSuccessSetup := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
		repo.On("GetAllByOwnerID", mock.Anything, "user-1").Return([]types.Organization{{ID: "org-owned", OwnerID: "user-1", Name: "Owned Org"}}, nil).Once()
		memberRepo.On("GetAllByUserID", mock.Anything, "user-1").Return([]types.OrganizationMember{{ID: "mem-owned", OrganizationID: "org-owned", UserID: "user-1", Role: "member"}, {ID: "mem-2", OrganizationID: "org-member", UserID: "user-1", Role: "member"}}, nil).Once()
		repo.On("Create", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
			return org != nil && org.OwnerID == "user-1" && org.Name == "Acme Labs" && org.Slug == "acme-labs" && string(org.Metadata) == "{}"
		})).Return(&types.Organization{ID: "org-2", OwnerID: "user-1", Name: "Acme Labs", Slug: "acme-labs", Metadata: json.RawMessage(`{}`)}, nil).Once()
		memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *types.OrganizationMember) bool {
			return member != nil && member.OrganizationID == "org-2" && member.UserID == "user-1" && member.Role == "member"
		})).Return(&types.OrganizationMember{ID: "mem-3", OrganizationID: "org-2", UserID: "user-1", Role: "member"}, nil).Once()
	}

	quotaExceededSetup := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
		repo.On("GetAllByOwnerID", mock.Anything, "user-1").Return([]types.Organization{{ID: "org-owned", OwnerID: "user-1", Name: "Owned Org"}}, nil).Once()
		memberRepo.On("GetAllByUserID", mock.Anything, "user-1").Return([]types.OrganizationMember{{ID: "mem-owned", OrganizationID: "org-owned", UserID: "user-1", Role: "member"}, {ID: "mem-2", OrganizationID: "org-member", UserID: "user-1", Role: "member"}}, nil).Once()
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
			expectErr:   internalerrors.ErrUnauthorized,
		},
		{
			name:        "missing role",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "Acme"},
			expectErr:   internalerrors.ErrUnprocessableEntity,
		},
		{
			name:        "bad request",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "", Role: "member"},
			expectErr:   internalerrors.ErrUnprocessableEntity,
		},
		{
			name:        "invalid role",
			actorUserID: "user-1",
			request:     types.CreateOrganizationRequest{Name: "Acme", Role: "ghost"},
			expectErr:   internalerrors.ErrUnprocessableEntity,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			request:        types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member", Metadata: json.RawMessage(`{"tier":"pro"}`)},
			setup:          successSetup,
			expectCalled:   true,
			expectReturned: "org-1",
		},
		{
			name:              "zero limit treated as unlimited",
			actorUserID:       "user-1",
			organizationLimit: &zeroLimit,
			request:           types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member", Metadata: json.RawMessage(`{"tier":"pro"}`)},
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
			org, err := svc.CreateOrganization(context.Background(), tt.actorUserID, tt.request)
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

func TestOrganizationService_GetAllOrganizations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		actorUserID string
		setup       func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr   error
		expectLen   int
	}{
		{
			name:        "unauthorized",
			actorUserID: "",
			expectErr:   internalerrors.ErrUnauthorized,
		},
		{
			name:        "success",
			actorUserID: "user-1",
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("GetAllByOwnerID", mock.Anything, "user-1").Return([]types.Organization{{ID: "org-1", OwnerID: "user-1", Name: "Acme"}}, nil).Once()
				memberRepo.On("GetAllByUserID", mock.Anything, "user-1").Return([]types.OrganizationMember{{ID: "mem-1", OrganizationID: "org-2", UserID: "user-1", Role: "member"}}, nil).Once()
				repo.On("GetByID", mock.Anything, "org-2").Return(&types.Organization{ID: "org-2", OwnerID: "owner-2", Name: "Platform"}, nil).Once()
			},
			expectLen: 2,
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
			organizations, err := svc.GetAllOrganizations(context.Background(), tt.actorUserID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, organizations, tt.expectLen)
			require.True(t, repo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
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
			expectErr: internalerrors.ErrForbidden,
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
			org, err := svc.GetOrganizationByID(context.Background(), tt.actorUserID, tt.organizationID)
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
			request:        types.UpdateOrganizationRequest{Name: "Acme Platform"},
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "unauthorized if no organization ID provided",
			actorUserID:    "user-1",
			organizationID: "",
			request:        types.UpdateOrganizationRequest{Name: "Acme Platform"},
			expectErr:      internalerrors.ErrUnauthorized,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: "Acme Platform"},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: internalerrors.ErrForbidden,
		},
		{
			name:           "success for member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: "Acme Platform"},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1", Name: "Acme", Slug: "acme"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.ID == "org-1" && org.Name == "Acme Platform" && org.Slug == "acme"
				})).Return(&types.Organization{ID: "org-1", OwnerID: "owner-1", Name: "Acme Platform", Slug: "acme"}, nil).Once()
			},
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.UpdateOrganizationRequest{Name: "Acme Platform"},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationHooks, serviceUtils *ServiceUtils) {
				repo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"}, nil).Once()
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
			org, err := svc.UpdateOrganization(context.Background(), tt.actorUserID, tt.organizationID, tt.request)
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
			err := svc.DeleteOrganization(context.Background(), tt.actorUserID, tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
