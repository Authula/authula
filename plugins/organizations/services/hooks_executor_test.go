package services

import (
	"context"
	"errors"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestServiceHookExecutor_NilConfigIsNoop(t *testing.T) {
	t.Parallel()

	executor := NewServiceHookExecutor(nil, nil)
	ctx := context.Background()
	actor := &models.Actor{ID: "user-1"}

	if err := executor.BeforeCreateOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeDeleteOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterDeleteOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeCreateOrganizationInvitation(ctx, actor, &types.OrganizationInvitation{ID: "inv-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganizationTeam(ctx, actor, &types.OrganizationTeam{ID: "team-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestServiceHookExecutor_NilExecutorIsNoop(t *testing.T) {
	t.Parallel()

	var executor *ServiceHookExecutor
	ctx := context.Background()
	actor := &models.Actor{ID: "user-1"}

	if err := executor.BeforeCreateOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganization(ctx, actor, &types.Organization{ID: "org-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestServiceHookExecutor_OrganizationCreateHooks(t *testing.T) {
	t.Parallel()

	var beforeCalled bool
	var afterCalled bool

	executor := NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{
		Organizations: &types.OrganizationServiceHooksConfig{
			BeforeCreate: func(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
				beforeCalled = true
				if organization == nil {
					return errors.New("organization is nil")
				}
				if organization.ID != "org-1" {
					t.Fatalf("unexpected organization ID: %s", organization.ID)
				}
				if actor == nil || actor.ID != "user-1" {
					t.Fatalf("unexpected actor: %+v", actor)
				}
				return nil
			},
			AfterCreate: func(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
				afterCalled = true
				if organization.ID != "org-1" {
					t.Fatalf("unexpected organization ID: %s", organization.ID)
				}
				if actor == nil || actor.ID != "user-1" {
					t.Fatalf("unexpected actor: %+v", actor)
				}
				return nil
			},
		},
	}, nil)

	ctx := context.Background()
	actor := &models.Actor{ID: "user-1"}
	organization := &types.Organization{ID: "org-1", Name: "Acme"}

	if err := executor.BeforeCreateOrganization(ctx, actor, organization); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterCreateOrganization(ctx, actor, organization); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeCalled {
		t.Fatal("expected BeforeCreate hook to be called")
	}
	if !afterCalled {
		t.Fatal("expected AfterCreate hook to be called")
	}
}

func TestServiceHookExecutor_OrganizationCreateHookError(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")
	executor := NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{
		Organizations: &types.OrganizationServiceHooksConfig{
			BeforeCreate: func(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
				return someErr
			},
		},
	}, nil)

	err := executor.BeforeCreateOrganization(context.Background(), &models.Actor{ID: "user-1"}, &types.Organization{ID: "org-1"})
	if !errors.Is(err, someErr) {
		t.Fatalf("expected someErr error, got %v", err)
	}
}

func TestServiceHookExecutor_MemberUpdateDeleteHooks(t *testing.T) {
	t.Parallel()

	var beforeUpdateCalled bool
	var afterUpdateCalled bool
	var beforeDeleteCalled bool
	var afterDeleteCalled bool

	executor := NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{
		Members: &types.OrganizationMemberServiceHooksConfig{
			BeforeUpdate: func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
				beforeUpdateCalled = true
				if member == nil || member.ID != "mem-1" {
					t.Fatalf("unexpected member in before update hook: %+v", member)
				}
				return nil
			},
			AfterUpdate: func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
				afterUpdateCalled = true
				if member.ID != "mem-1" {
					t.Fatalf("unexpected member in after update hook: %+v", member)
				}
				return nil
			},
			BeforeDelete: func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
				beforeDeleteCalled = true
				if member == nil || member.ID != "mem-1" {
					t.Fatalf("unexpected member in before delete hook: %+v", member)
				}
				return nil
			},
			AfterDelete: func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
				afterDeleteCalled = true
				if member.ID != "mem-1" {
					t.Fatalf("unexpected member in after delete hook: %+v", member)
				}
				return nil
			},
		},
	}, nil)

	ctx := context.Background()
	actor := &models.Actor{ID: "user-1"}
	member := &types.OrganizationMember{ID: "mem-1", Role: "member"}

	if err := executor.BeforeUpdateOrganizationMember(ctx, actor, member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterUpdateOrganizationMember(ctx, actor, member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeDeleteOrganizationMember(ctx, actor, member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterDeleteOrganizationMember(ctx, actor, member); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeUpdateCalled || !afterUpdateCalled || !beforeDeleteCalled || !afterDeleteCalled {
		t.Fatal("expected member update and delete hooks to be called")
	}
}

func TestServiceHookExecutor_PluginRegistryAccessibleInHook(t *testing.T) {
	t.Parallel()

	var receivedPluginRegistry models.PluginRegistry
	pluginRegistry := &internaltests.MockPluginRegistry{}

	executor := NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{
		Organizations: &types.OrganizationServiceHooksConfig{
			BeforeCreate: func(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
				reg := models.GetPluginRegistryFromContext(ctx)
				if reg == nil {
					return errors.New("plugin registry not found in context")
				}
				receivedPluginRegistry = reg
				return nil
			},
		},
	}, pluginRegistry)

	err := executor.BeforeCreateOrganization(context.Background(), &models.Actor{ID: "user-1"}, &types.Organization{ID: "org-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if receivedPluginRegistry != pluginRegistry {
		t.Fatal("expected the injected plugin registry to be accessible from context")
	}
}
