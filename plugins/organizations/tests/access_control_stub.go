package tests

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	rootservices "github.com/Authula/authula/services"
)

type AccessControlServiceStub struct {
	RoleWeights     map[string]int
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

func (s *AccessControlServiceStub) ValidateRoleAssignment(ctx context.Context, roleName string, assignerUserID *string) (bool, error) {
	_ = ctx
	if s != nil && s.Err != nil {
		return false, s.Err
	}

	roleWeight, ok := s.roleWeight(roleName)
	if !ok {
		return false, nil
	}
	if s == nil || len(s.AssignerWeights) == 0 {
		return true, nil
	}

	if assignerUserID == nil || *assignerUserID == "" {
		return false, nil
	}

	assignerWeight, ok := s.assignerWeight(*assignerUserID)
	if !ok {
		return false, nil
	}

	if roleWeight > assignerWeight {
		return false, internalerrors.ErrForbidden
	}

	return true, nil
}

func (s *AccessControlServiceStub) roleWeight(roleName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	weight, ok := s.RoleWeights[roleName]
	return weight, ok
}

func (s *AccessControlServiceStub) assignerWeight(userID string) (int, bool) {
	if s == nil {
		return 0, false
	}
	weight, ok := s.AssignerWeights[userID]
	return weight, ok
}
