package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

// memberRoleWeightAccessControl overrides role weight lookups per role, which
// the shared stub cannot do because it fails every lookup when Err is set.
type memberRoleWeightAccessControl struct {
	*orgtests.AccessControlServiceStub
	errByRole map[string]error
}

func (s *memberRoleWeightAccessControl) GetRoleWeightByName(ctx context.Context, roleName string) (int, error) {
	if err, ok := s.errByRole[roleName]; ok {
		return 0, err
	}
	return s.AccessControlServiceStub.GetRoleWeightByName(ctx, roleName)
}

func newMemberHooksExecutor(configure func(*orgtypes.OrganizationMemberServiceHooks)) *ServiceHookExecutor {
	hooks := &orgtypes.OrganizationMemberServiceHooks{}
	if configure != nil {
		configure(hooks)
	}
	return NewServiceHookExecutor(&orgtypes.OrganizationsServiceHooksConfig{Members: hooks}, nil, nil)
}

func actorMember(memberRepo *orgtests.MockOrganizationMemberRepository, organizationID, userID, role string) {
	memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, organizationID, userID).
		Return(&orgtypes.OrganizationMember{ID: "mem-actor", OrganizationID: organizationID, UserID: userID, Role: role}, nil).
		Once()
}

type memberHooksFixture struct {
	orgRepo     *orgtests.MockOrganizationRepository
	memberRepo  *orgtests.MockOrganizationMemberRepository
	userSvc     *internaltests.MockUserService
	executor    *ServiceHookExecutor
	beforeActor string
	afterActor  string
}

func newMemberHooksFixture() *memberHooksFixture {
	return &memberHooksFixture{
		orgRepo:    &orgtests.MockOrganizationRepository{},
		memberRepo: &orgtests.MockOrganizationMemberRepository{},
		userSvc:    &internaltests.MockUserService{},
	}
}

func (f *memberHooksFixture) service() *organizationMemberService {
	return newOrganizationMemberServiceWithHooks(f.userSvc, f.orgRepo, f.memberRepo, f.executor)
}

func TestOrganizationMemberService_AddMemberHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before create hook failed")
	errAfterHook := errors.New("after create hook failed")

	expectBase := func(f *memberHooksFixture) {
		f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(f.memberRepo, "org-1", "user-1", "admin")
		f.userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
		f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
	}

	tests := []struct {
		name   string
		setup  func(*memberHooksFixture)
		assert func(*testing.T, *memberHooksFixture, *orgtypes.OrganizationMember, error)
	}{
		{
			name: "constructor wires the hook executor and the hooks fire",
			setup: func(f *memberHooksFixture) {
				expectBase(f)
				f.memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *orgtypes.OrganizationMember) bool {
					return member != nil && member.UserID == "user-2" && member.Role == "member"
				})).Return(&orgtypes.OrganizationMember{ID: "mem-2", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterBeforeCreate(func(_ context.Context, actor *models.Actor, _ *orgtypes.OrganizationMember) error {
						f.beforeActor = actor.ID
						return nil
					})
					hooks.RegisterAfterCreate(func(_ context.Context, actor *models.Actor, _ *orgtypes.OrganizationMember) error {
						f.afterActor = actor.ID
						return nil
					})
				})
			},
			assert: func(t *testing.T, f *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.NoError(t, err)
				require.Equal(t, "mem-2", member.ID)
				require.Equal(t, "user-1", f.beforeActor)
				require.Equal(t, "user-1", f.afterActor)
			},
		},
		{
			name: "before create hook error is propagated",
			setup: func(f *memberHooksFixture) {
				expectBase(f)
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterBeforeCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errBeforeHook
					})
				})
			},
			assert: func(t *testing.T, f *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				require.Nil(t, member)
				f.memberRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			},
		},
		{
			name: "after create hook error is propagated",
			setup: func(f *memberHooksFixture) {
				expectBase(f)
				f.memberRepo.On("Create", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-2"}, nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.ErrorIs(t, err, errAfterHook)
				require.Nil(t, member)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newMemberHooksFixture()
			tc.setup(f)

			member, err := f.service().AddMember(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})

			tc.assert(t, f, member, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.userSvc.AssertExpectations(t)
		})
	}
}

func newOrganizationMemberServiceWithHooks(userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, executor *ServiceHookExecutor) *organizationMemberService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	return NewOrganizationMemberService(userSvc, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil, &orgtests.MockTxRunner{}, serviceUtils, executor)
}

func TestOrganizationMemberService_UpdateMemberSelfAndHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before update hook failed")
	errAfterHook := errors.New("after update hook failed")

	tests := []struct {
		name   string
		setup  func(*memberHooksFixture)
		assert func(*testing.T, *memberHooksFixture, *orgtypes.OrganizationMember, error)
	}{
		{
			name: "actors cannot change their own role",
			setup: func(f *memberHooksFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				actorMember(f.memberRepo, "org-1", "user-1", "admin")
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()
			},
			assert: func(t *testing.T, f *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.ErrorIs(t, err, coreerrors.ErrForbidden)
				require.Nil(t, member)
				f.memberRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name: "before update hook error is propagated",
			setup: func(f *memberHooksFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				actorMember(f.memberRepo, "org-1", "user-1", "admin")
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errBeforeHook
					})
				})
			},
			assert: func(t *testing.T, f *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				require.Nil(t, member)
				f.memberRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name: "after update hook error is propagated",
			setup: func(f *memberHooksFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				actorMember(f.memberRepo, "org-1", "user-1", "admin")
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				f.memberRepo.On("Update", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-1", UserID: "user-2", Role: "admin"}, nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *memberHooksFixture, member *orgtypes.OrganizationMember, err error) {
				require.ErrorIs(t, err, errAfterHook)
				require.Nil(t, member)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newMemberHooksFixture()
			tc.setup(f)

			member, err := f.service().UpdateMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1", orgtypes.UpdateOrganizationMemberRequest{Role: "admin"})

			tc.assert(t, f, member, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.userSvc.AssertExpectations(t)
		})
	}
}

func TestOrganizationMemberService_RemoveMemberHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before delete hook failed")
	errAfterHook := errors.New("after delete hook failed")

	tests := []struct {
		name   string
		setup  func(*memberHooksFixture) *models.Actor
		assert func(*testing.T, *memberHooksFixture, error)
	}{
		{
			name: "machine actors cannot remove members without a member record",
			setup: func(f *memberHooksFixture) *models.Actor {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				return &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}}
			},
			assert: func(t *testing.T, f *memberHooksFixture, err error) {
				require.ErrorIs(t, err, coreerrors.ErrForbidden)
				f.memberRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			name: "before delete hook error is propagated",
			setup: func(f *memberHooksFixture) *models.Actor {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				actorMember(f.memberRepo, "org-1", "user-1", "admin")
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterBeforeDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errBeforeHook
					})
				})
				return orgtests.Actor("user-1")
			},
			assert: func(t *testing.T, f *memberHooksFixture, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				f.memberRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			name: "after delete hook error is propagated",
			setup: func(f *memberHooksFixture) *models.Actor {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				actorMember(f.memberRepo, "org-1", "user-1", "admin")
				f.memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				f.memberRepo.On("Delete", mock.Anything, "mem-1").Return(nil).Once()
				f.executor = newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
					hooks.RegisterAfterDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
						return errAfterHook
					})
				})
				return orgtests.Actor("user-1")
			},
			assert: func(t *testing.T, _ *memberHooksFixture, err error) {
				require.ErrorIs(t, err, errAfterHook)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newMemberHooksFixture()
			actor := tc.setup(f)

			err := f.service().RemoveMember(context.Background(), actor, "org-1", "mem-1")

			tc.assert(t, f, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.userSvc.AssertExpectations(t)
		})
	}
}

func TestAuthorizeRemoveMemberWeight(t *testing.T) {
	t.Parallel()

	errBackend := errors.New("access control unavailable")
	actor := &orgtypes.OrganizationMember{ID: "mem-actor", UserID: "user-1", Role: "admin"}
	target := &orgtypes.OrganizationMember{ID: "mem-target", UserID: "user-2", Role: "member"}

	tests := []struct {
		name          string
		accessControl func() *memberRoleWeightAccessControl
		target        *orgtypes.OrganizationMember
		expectErr     error
	}{
		{
			name: "unknown actor role is forbidden",
			accessControl: func() *memberRoleWeightAccessControl {
				return &memberRoleWeightAccessControl{
					AccessControlServiceStub: orgtests.NewAccessControlServiceStubWithWeights(map[string]int{"member": 10}, nil),
					errByRole:                map[string]error{},
				}
			},
			target:    target,
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name: "actor role backend error is returned",
			accessControl: func() *memberRoleWeightAccessControl {
				return &memberRoleWeightAccessControl{
					AccessControlServiceStub: orgtests.NewAccessControlServiceStub(),
					errByRole:                map[string]error{"admin": errBackend},
				}
			},
			target:    target,
			expectErr: errBackend,
		},
		{
			name: "unknown target role is forbidden",
			accessControl: func() *memberRoleWeightAccessControl {
				return &memberRoleWeightAccessControl{
					AccessControlServiceStub: orgtests.NewAccessControlServiceStubWithWeights(map[string]int{"admin": 20}, nil),
					errByRole:                map[string]error{},
				}
			},
			target:    target,
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name: "target role backend error is returned",
			accessControl: func() *memberRoleWeightAccessControl {
				return &memberRoleWeightAccessControl{
					AccessControlServiceStub: orgtests.NewAccessControlServiceStub(),
					errByRole:                map[string]error{"member": errBackend},
				}
			},
			target:    target,
			expectErr: errBackend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := authorizeRemoveMemberWeight(context.Background(), tt.accessControl(), actor, tt.target)
			require.ErrorIs(t, err, tt.expectErr)
		})
	}
}
