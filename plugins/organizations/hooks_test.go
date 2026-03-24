package organizations

import (
	"errors"
	"testing"

	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationsHookExecutor_NilHooksAreNoop(t *testing.T) {
	t.Parallel()

	executor := NewOrganizationsHookExecutor(nil)

	if err := executor.BeforeCreateOrganization(&types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganization(types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeCreateOrganizationInvitation(&types.OrganizationInvitation{ID: "inv-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganizationTeam(types.OrganizationTeam{ID: "team-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestOrganizationsHookExecutor_OrganizationCreateHooks(t *testing.T) {
	t.Parallel()

	var beforeCalled bool
	var afterCalled bool

	executor := NewOrganizationsHookExecutor(&types.OrganizationsDatabaseHooksConfig{
		Organizations: &types.OrganizationDatabaseHooksConfig{
			BeforeCreate: func(organization *types.Organization) error {
				beforeCalled = true
				if organization == nil {
					return errors.New("organization is nil")
				}
				if organization.ID != "org-1" {
					t.Fatalf("unexpected organization ID: %s", organization.ID)
				}
				return nil
			},
			AfterCreate: func(organization types.Organization) error {
				afterCalled = true
				if organization.ID != "org-1" {
					t.Fatalf("unexpected organization ID: %s", organization.ID)
				}
				return nil
			},
		},
	})

	organization := &types.Organization{ID: "org-1", Name: "Acme"}
	if err := executor.BeforeCreateOrganization(organization); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganization(*organization); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeCalled {
		t.Fatal("expected BeforeCreate hook to be called")
	}
	if !afterCalled {
		t.Fatal("expected AfterCreate hook to be called")
	}
}

func TestOrganizationsHookExecutor_OrganizationCreateHookError(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")
	executor := NewOrganizationsHookExecutor(&types.OrganizationsDatabaseHooksConfig{
		Organizations: &types.OrganizationDatabaseHooksConfig{
			BeforeCreate: func(organization *types.Organization) error {
				return someErr
			},
		},
	})

	err := executor.BeforeCreateOrganization(&types.Organization{ID: "org-1"})
	if !errors.Is(err, someErr) {
		t.Fatalf("expected someErr error, got %v", err)
	}
}

func TestOrganizationsHookExecutor_MemberUpdateDeleteHooks(t *testing.T) {
	t.Parallel()

	var beforeUpdateCalled bool
	var afterUpdateCalled bool
	var beforeDeleteCalled bool
	var afterDeleteCalled bool

	executor := NewOrganizationsHookExecutor(&types.OrganizationsDatabaseHooksConfig{
		Members: &types.OrganizationMemberDatabaseHooksConfig{
			BeforeUpdate: func(member *types.OrganizationMember) error {
				beforeUpdateCalled = true
				if member == nil || member.ID != "mem-1" {
					t.Fatalf("unexpected member in before update hook: %+v", member)
				}
				return nil
			},
			AfterUpdate: func(member types.OrganizationMember) error {
				afterUpdateCalled = true
				if member.ID != "mem-1" {
					t.Fatalf("unexpected member in after update hook: %+v", member)
				}
				return nil
			},
			BeforeDelete: func(member *types.OrganizationMember) error {
				beforeDeleteCalled = true
				if member == nil || member.ID != "mem-1" {
					t.Fatalf("unexpected member in before delete hook: %+v", member)
				}
				return nil
			},
			AfterDelete: func(member types.OrganizationMember) error {
				afterDeleteCalled = true
				if member.ID != "mem-1" {
					t.Fatalf("unexpected member in after delete hook: %+v", member)
				}
				return nil
			},
		},
	})

	member := &types.OrganizationMember{ID: "mem-1", Role: "member"}
	if err := executor.BeforeUpdateOrganizationMember(member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterUpdateOrganizationMember(*member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeDeleteOrganizationMember(member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterDeleteOrganizationMember(*member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeUpdateCalled || !afterUpdateCalled || !beforeDeleteCalled || !afterDeleteCalled {
		t.Fatal("expected member update and delete hooks to be called")
	}
}
