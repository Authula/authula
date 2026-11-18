package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	coretypes "github.com/Authula/authula/core/types"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

// roleExistsErrorAccessControl forces RoleExists to fail. The shared stub always
// succeeds because it checks a fixed role map.
type roleExistsErrorAccessControl struct {
	*orgtests.AccessControlServiceStub
	err error
}

func (s *roleExistsErrorAccessControl) RoleExists(context.Context, string) (bool, error) {
	return false, s.err
}

func newOrganizationHooksExecutor(configure func(*types.OrganizationServiceHooks)) *ServiceHookExecutor {
	hooks := &types.OrganizationServiceHooks{}
	if configure != nil {
		configure(hooks)
	}
	return NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{Organizations: hooks}, nil, nil)
}

func TestOrganizationService_CreateOrganizationErrorBranches(t *testing.T) {
	t.Parallel()

	errRoleExists := errors.New("role existence lookup failed")
	errRolePermissions := errors.New("role permissions lookup failed")
	errRepoCreate := errors.New("organization create failed")
	errMemberCreate := errors.New("member create failed")
	errBeforeHook := errors.New("before create hook failed")
	errAfterHook := errors.New("after create hook failed")

	successRepos := func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
		repo.On("Create", mock.Anything, mock.Anything).Return(&types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"}, nil).Once()
		memberRepo.On("Create", mock.Anything, mock.Anything).Return(&types.OrganizationMember{ID: "mem-1"}, nil).Once()
	}

	tests := []struct {
		name          string
		request       types.CreateOrganizationRequest
		accessControl rootservices.AccessControlService
		setup         func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationMemberRepository)
		configure     func(*types.OrganizationServiceHooks)
		expectErr     error
		expectRepoHit bool
	}{
		{
			name:    "role existence error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			accessControl: &roleExistsErrorAccessControl{
				AccessControlServiceStub: orgtests.NewAccessControlServiceStub(),
				err:                      errRoleExists,
			},
			expectErr: errRoleExists,
		},
		{
			name:    "role permission lookup error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			accessControl: func() rootservices.AccessControlService {
				stub := orgtests.NewAccessControlServiceStub()
				stub.Err = errRolePermissions
				return stub
			}(),
			expectErr: errRolePermissions,
		},
		{
			name:    "explicit slug is used as provided",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member", Slug: new("custom-slug")},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("Create", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.Slug == "custom-slug" && org.Name == "Acme Inc"
				})).Return(&types.Organization{ID: "org-2", Slug: "custom-slug"}, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.Anything).Return(&types.OrganizationMember{ID: "mem-2"}, nil).Once()
			},
			expectRepoHit: true,
		},
		{
			name:      "name without letters or digits cannot produce a slug",
			request:   types.CreateOrganizationRequest{Name: "!!!", Role: "member"},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:    "before create hook error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterBeforeCreate(func(context.Context, *models.Actor, *types.Organization) error {
					return errBeforeHook
				})
			},
			expectErr: errBeforeHook,
		},
		{
			name:    "organization create error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			setup: func(repo *orgtests.MockOrganizationRepository, _ *orgtests.MockOrganizationMemberRepository) {
				repo.On("Create", mock.Anything, mock.Anything).Return(nil, errRepoCreate).Once()
			},
			expectErr:     errRepoCreate,
			expectRepoHit: true,
		},
		{
			name:    "member create error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			setup: func(repo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				repo.On("Create", mock.Anything, mock.Anything).Return(&types.Organization{ID: "org-1"}, nil).Once()
				memberRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errMemberCreate).Once()
			},
			expectErr:     errMemberCreate,
			expectRepoHit: true,
		},
		{
			name:    "after create hook error is propagated",
			request: types.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"},
			setup:   successRepos,
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterAfterCreate(func(context.Context, *models.Actor, *types.Organization) error {
					return errAfterHook
				})
			},
			expectErr:     errAfterHook,
			expectRepoHit: true,
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
			accessControl := tt.accessControl
			if accessControl == nil {
				accessControl = orgtests.NewAccessControlServiceStub()
			}

			serviceUtils := &ServiceUtils{orgRepo: repo, orgMemberRepo: memberRepo}
			svc := NewOrganizationService(repo, memberRepo, serviceUtils, accessControl, nil, &orgtests.MockTxRunner{}, newOrganizationHooksExecutor(tt.configure))

			org, err := svc.CreateOrganization(context.Background(), orgtests.Actor("user-1"), tt.request)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, org)
			} else {
				require.NoError(t, err)
				require.NotNil(t, org)
			}

			if tt.expectRepoHit {
				repo.AssertExpectations(t)
				memberRepo.AssertExpectations(t)
			}
		})
	}
}

func TestOrganizationService_UpdateOrganizationErrorBranches(t *testing.T) {
	t.Parallel()

	errUpdate := errors.New("update failed")
	errBeforeHook := errors.New("before update hook failed")
	errAfterHook := errors.New("after update hook failed")

	existing := func(slug string) *types.Organization {
		return &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: slug}
	}

	tests := []struct {
		name         string
		organization *types.Organization
		request      types.UpdateOrganizationRequest
		setup        func(*orgtests.MockOrganizationRepository)
		configure    func(*types.OrganizationServiceHooks)
		expectErr    error
		expectUpdate bool
	}{
		{
			name:         "empty name is rejected",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Name: new("")},
			expectErr:    coreerrors.ErrUnprocessableEntity,
		},
		{
			name:         "explicit slug overrides the stored slug",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Slug: new("new-slug")},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.Slug == "new-slug"
				})).Return(&types.Organization{ID: "org-1", Slug: "new-slug"}, nil).Once()
			},
			expectUpdate: true,
		},
		{
			name:         "slug is derived from the new name when the stored slug is empty",
			organization: existing(""),
			request:      types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.Slug == "acme-platform"
				})).Return(&types.Organization{ID: "org-1", Slug: "acme-platform"}, nil).Once()
			},
			expectUpdate: true,
		},
		{
			name:         "name without letters or digits cannot produce a slug",
			organization: existing(""),
			request:      types.UpdateOrganizationRequest{Name: new("!!!")},
			expectErr:    coreerrors.ErrUnprocessableEntity,
		},
		{
			name:         "logo value is set",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Logo: coretypes.Optional[*string]{Present: true, Value: new("https://cdn.example.com/logo.svg")}},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.Logo != nil && *org.Logo == "https://cdn.example.com/logo.svg"
				})).Return(&types.Organization{ID: "org-1", Logo: new("https://cdn.example.com/logo.svg")}, nil).Once()
			},
			expectUpdate: true,
		},
		{
			name:         "metadata is replaced",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Metadata: map[string]any{"tier": "pro"}},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(org *types.Organization) bool {
					return org != nil && org.Metadata["tier"] == "pro"
				})).Return(&types.Organization{ID: "org-1", Metadata: map[string]any{"tier": "pro"}}, nil).Once()
			},
			expectUpdate: true,
		},
		{
			name:         "before update hook error is propagated",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterBeforeUpdate(func(context.Context, *models.Actor, *types.Organization) error {
					return errBeforeHook
				})
			},
			expectErr: errBeforeHook,
		},
		{
			name:         "update error is propagated",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.Anything).Return(nil, errUpdate).Once()
			},
			expectErr:    errUpdate,
			expectUpdate: true,
		},
		{
			name:         "after update hook error is propagated",
			organization: existing("acme"),
			request:      types.UpdateOrganizationRequest{Name: new("Acme Platform")},
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Update", mock.Anything, mock.Anything).Return(&types.Organization{ID: "org-1", Name: "Acme Platform", Slug: "acme"}, nil).Once()
			},
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterAfterUpdate(func(context.Context, *models.Actor, *types.Organization) error {
					return errAfterHook
				})
			},
			expectErr:    errAfterHook,
			expectUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			repo.On("GetByID", mock.Anything, "org-1").Return(tt.organization, nil).Once()
			if tt.setup != nil {
				tt.setup(repo)
			}

			serviceUtils := &ServiceUtils{orgRepo: repo}
			svc := NewOrganizationService(repo, nil, serviceUtils, nil, nil, nil, newOrganizationHooksExecutor(tt.configure))

			org, err := svc.UpdateOrganization(context.Background(), orgtests.Actor("user-1"), "org-1", tt.request)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, org)
			} else {
				require.NoError(t, err)
				require.NotNil(t, org)
			}

			if tt.expectUpdate {
				repo.AssertExpectations(t)
			}
		})
	}
}

func TestOrganizationService_DeleteOrganizationErrorBranches(t *testing.T) {
	t.Parallel()

	errDelete := errors.New("delete failed")
	errBeforeHook := errors.New("before delete hook failed")
	errAfterHook := errors.New("after delete hook failed")

	owned := func() *types.Organization {
		return &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"}
	}

	tests := []struct {
		name         string
		organization *types.Organization
		setup        func(*orgtests.MockOrganizationRepository)
		configure    func(*types.OrganizationServiceHooks)
		expectErr    error
		expectDelete bool
	}{
		{
			name:         "owner authorization error is propagated",
			organization: &types.Organization{ID: "org-1", OwnerID: "owner-1"},
			expectErr:    coreerrors.ErrForbidden,
		},
		{
			name:         "before delete hook error is propagated",
			organization: owned(),
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterBeforeDelete(func(context.Context, *models.Actor, *types.Organization) error {
					return errBeforeHook
				})
			},
			expectErr: errBeforeHook,
		},
		{
			name:         "delete error is propagated",
			organization: owned(),
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Delete", mock.Anything, "org-1").Return(errDelete).Once()
			},
			expectErr:    errDelete,
			expectDelete: true,
		},
		{
			name:         "after delete hook error is propagated",
			organization: owned(),
			setup: func(repo *orgtests.MockOrganizationRepository) {
				repo.On("Delete", mock.Anything, "org-1").Return(nil).Once()
			},
			configure: func(h *types.OrganizationServiceHooks) {
				h.RegisterAfterDelete(func(context.Context, *models.Actor, *types.Organization) error {
					return errAfterHook
				})
			},
			expectErr:    errAfterHook,
			expectDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &orgtests.MockOrganizationRepository{}
			repo.On("GetByID", mock.Anything, "org-1").Return(tt.organization, nil).Once()
			if tt.setup != nil {
				tt.setup(repo)
			}

			serviceUtils := &ServiceUtils{orgRepo: repo}
			svc := NewOrganizationService(repo, nil, serviceUtils, nil, nil, nil, newOrganizationHooksExecutor(tt.configure))

			err := svc.DeleteOrganization(context.Background(), orgtests.Actor("user-1"), "org-1")
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}

			if tt.expectDelete {
				repo.AssertExpectations(t)
			}
		})
	}
}
