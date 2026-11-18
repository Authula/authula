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

func TestOrganizationMemberService_AddMemberHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before create hook failed")
	errAfterHook := errors.New("after create hook failed")

	expectBase := func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, userSvc *internaltests.MockUserService) {
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		userSvc.On("GetByID", mock.Anything, "user-2").Return(&models.User{ID: "user-2", Email: "user2@example.com"}, nil).Once()
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
	}

	t.Run("constructor wires the hook executor and the hooks fire", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		userSvc := &internaltests.MockUserService{}
		expectBase(orgRepo, memberRepo, userSvc)
		memberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *orgtypes.OrganizationMember) bool {
			return member != nil && member.UserID == "user-2" && member.Role == "member"
		})).Return(&orgtypes.OrganizationMember{ID: "mem-2", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()

		var beforeActor, afterActor string
		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterBeforeCreate(func(_ context.Context, actor *models.Actor, _ *orgtypes.OrganizationMember) error {
				beforeActor = actor.ID
				return nil
			})
			hooks.RegisterAfterCreate(func(_ context.Context, actor *models.Actor, _ *orgtypes.OrganizationMember) error {
				afterActor = actor.ID
				return nil
			})
		})

		serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
		svc := NewOrganizationMemberService(userSvc, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil, &orgtests.MockTxRunner{}, serviceUtils, executor)

		member, err := svc.AddMember(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})

		require.NoError(t, err)
		require.Equal(t, "mem-2", member.ID)
		require.Equal(t, "user-1", beforeActor)
		require.Equal(t, "user-1", afterActor)
		orgRepo.AssertExpectations(t)
		memberRepo.AssertExpectations(t)
		userSvc.AssertExpectations(t)
	})

	t.Run("before create hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		userSvc := &internaltests.MockUserService{}
		expectBase(orgRepo, memberRepo, userSvc)

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterBeforeCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(userSvc, orgRepo, memberRepo, executor)
		member, err := svc.AddMember(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})

		require.ErrorIs(t, err, errBeforeHook)
		require.Nil(t, member)
		memberRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("after create hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		userSvc := &internaltests.MockUserService{}
		expectBase(orgRepo, memberRepo, userSvc)
		memberRepo.On("Create", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-2"}, nil).Once()

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errAfterHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(userSvc, orgRepo, memberRepo, executor)
		member, err := svc.AddMember(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})

		require.ErrorIs(t, err, errAfterHook)
		require.Nil(t, member)
		memberRepo.AssertExpectations(t)
	})
}

func newOrganizationMemberServiceWithHooks(userSvc *internaltests.MockUserService, orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, executor *ServiceHookExecutor) *organizationMemberService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	return NewOrganizationMemberService(userSvc, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil, &orgtests.MockTxRunner{}, serviceUtils, executor)
}

func TestOrganizationMemberService_UpdateMemberSelfAndHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before update hook failed")
	errAfterHook := errors.New("after update hook failed")

	t.Run("actors cannot change their own role", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Once()

		svc := newTestOrganizationMemberService(&internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
		member, err := svc.UpdateMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1", orgtypes.UpdateOrganizationMemberRequest{Role: "admin"})

		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		require.Nil(t, member)
		memberRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("before update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(&internaltests.MockUserService{}, orgRepo, memberRepo, executor)
		member, err := svc.UpdateMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1", orgtypes.UpdateOrganizationMemberRequest{Role: "admin"})

		require.ErrorIs(t, err, errBeforeHook)
		require.Nil(t, member)
		memberRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("after update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
		memberRepo.On("Update", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-1", UserID: "user-2", Role: "admin"}, nil).Once()

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errAfterHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(&internaltests.MockUserService{}, orgRepo, memberRepo, executor)
		member, err := svc.UpdateMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1", orgtypes.UpdateOrganizationMemberRequest{Role: "admin"})

		require.ErrorIs(t, err, errAfterHook)
		require.Nil(t, member)
		memberRepo.AssertExpectations(t)
	})
}

func TestOrganizationMemberService_RemoveMemberHooks(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before delete hook failed")
	errAfterHook := errors.New("after delete hook failed")

	t.Run("machine actors cannot remove members without a member record", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()

		actor := &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}}
		svc := newTestOrganizationMemberService(&internaltests.MockUserService{}, orgtests.NewAccessControlServiceStub(), orgRepo, memberRepo, nil)
		err := svc.RemoveMember(context.Background(), actor, "org-1", "mem-1")

		require.ErrorIs(t, err, coreerrors.ErrForbidden)
		memberRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("before delete hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterBeforeDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(&internaltests.MockUserService{}, orgRepo, memberRepo, executor)
		err := svc.RemoveMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1")

		require.ErrorIs(t, err, errBeforeHook)
		memberRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("after delete hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		actorMember(memberRepo, "org-1", "user-1", "admin")
		memberRepo.On("GetByID", mock.Anything, "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
		memberRepo.On("Delete", mock.Anything, "mem-1").Return(nil).Once()

		executor := newMemberHooksExecutor(func(hooks *orgtypes.OrganizationMemberServiceHooks) {
			hooks.RegisterAfterDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationMember) error {
				return errAfterHook
			})
		})

		svc := newOrganizationMemberServiceWithHooks(&internaltests.MockUserService{}, orgRepo, memberRepo, executor)
		err := svc.RemoveMember(context.Background(), orgtests.Actor("user-1"), "org-1", "mem-1")

		require.ErrorIs(t, err, errAfterHook)
		memberRepo.AssertExpectations(t)
	})
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
