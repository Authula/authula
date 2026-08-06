package tests

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	rootservices "github.com/Authula/authula/services"
)

type AccessControlServiceStub struct {
	RoleWeights     map[string]int
	RolePermissions map[string][]string
	AssignerWeights map[string]int
	Err             error
}

var _ rootservices.AccessControlService = (*AccessControlServiceStub)(nil)

func NewAccessControlServiceStub() *AccessControlServiceStub {
	return NewAccessControlServiceStubWithWeights(nil, nil)
}

func NewAccessControlServiceStubWithWeights(roleWeights, assignerWeights map[string]int) *AccessControlServiceStub {
	if roleWeights == nil {
		roleWeights = map[string]int{
			"member":  10,
			"admin":   20,
			"manager": 30,
		}
	}
	if assignerWeights == nil {
		assignerWeights = map[string]int{}
	}

	return &AccessControlServiceStub{
		RoleWeights:     roleWeights,
		RolePermissions: map[string][]string{},
		AssignerWeights: assignerWeights,
	}
}

func NewRoleHierarchyAccessControlServiceStub(roleWeights, assignerWeights map[string]int) *AccessControlServiceStub {
	return NewAccessControlServiceStubWithWeights(roleWeights, assignerWeights)
}

func (s *AccessControlServiceStub) RoleExists(ctx context.Context, roleName string) (bool, error) {
	_ = ctx
	_, ok := s.roleWeight(roleName)
	if !ok {
		return false, nil
	}

	return true, nil
}

func (s *AccessControlServiceStub) GetRolePermissionsByName(ctx context.Context, roleName string) ([]string, error) {
	_ = ctx
	if s != nil && s.Err != nil {
		return nil, s.Err
	}

	if perms, ok := s.RolePermissions[roleName]; ok {
		return perms, nil
	}
	if _, ok := s.roleWeight(roleName); ok {
		return []string{"*"}, nil
	}

	return nil, coreerrors.ErrNotFound
}

func (s *AccessControlServiceStub) GetRoleWeightByName(ctx context.Context, roleName string) (int, error) {
	_ = ctx
	if s != nil && s.Err != nil {
		return 0, s.Err
	}

	weight, ok := s.roleWeight(roleName)
	if !ok {
		return 0, coreerrors.ErrNotFound
	}

	return weight, nil
}

func (s *AccessControlServiceStub) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	_ = ctx
	_ = permissionKeys
	if s != nil && s.Err != nil {
		return s.Err
	}

	return nil
}

func (s *AccessControlServiceStub) EnsurePermissions(ctx context.Context, permissions []rootservices.PermissionDefinition) error {
	_ = ctx
	_ = permissions
	if s != nil && s.Err != nil {
		return s.Err
	}

	return nil
}

func (s *AccessControlServiceStub) roleWeight(roleName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	weight, ok := s.RoleWeights[roleName]
	return weight, ok
}
