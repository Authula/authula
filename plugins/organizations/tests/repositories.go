package tests

import (
	"context"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
)

type MockOrganizationRepository struct {
	mock.Mock
	Hooks repositories.OrganizationHookExecutor
	byID  map[string]*types.Organization
}

func mockResultIsNil(value any) bool {
	if value == nil {
		return true
	}
	result := reflect.ValueOf(value)
	switch result.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return result.IsNil()
	default:
		return false
	}
}

func (m *MockOrganizationRepository) SetOrganizationHooks(hooks repositories.OrganizationHookExecutor) {
	m.Hooks = hooks
}

func (m *MockOrganizationRepository) Create(ctx context.Context, organization *types.Organization) (*types.Organization, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeCreateOrganization(organization); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, organization)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	created := args.Get(0).(*types.Organization)
	if m.Hooks != nil {
		if err := m.Hooks.AfterCreateOrganization(*created); err != nil {
			return nil, err
		}
	}
	return created, args.Error(1)
}

func (m *MockOrganizationRepository) GetByID(ctx context.Context, organizationID string) (*types.Organization, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	organization := args.Get(0).(*types.Organization)
	if m.byID == nil {
		m.byID = map[string]*types.Organization{}
	}
	m.byID[organizationID] = organization
	return organization, args.Error(1)
}

func (m *MockOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetAllByOwnerID(ctx context.Context, ownerID string) ([]types.Organization, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Update(ctx context.Context, organization *types.Organization) (*types.Organization, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeUpdateOrganization(organization); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, organization)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	updated := args.Get(0).(*types.Organization)
	if m.Hooks != nil {
		if err := m.Hooks.AfterUpdateOrganization(*updated); err != nil {
			return nil, err
		}
	}
	return updated, args.Error(1)
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, organizationID string) error {
	if m.Hooks != nil {
		organization := m.byID[organizationID]
		if organization == nil {
			var err error
			organization, err = m.GetByID(ctx, organizationID)
			if err != nil {
				return err
			}
		}
		if organization == nil {
			return nil
		}
		if err := m.Hooks.BeforeDeleteOrganization(organization); err != nil {
			return err
		}
		if err := m.Called(ctx, organizationID).Error(0); err != nil {
			return err
		}
		return m.Hooks.AfterDeleteOrganization(*organization)
	}
	return m.Called(ctx, organizationID).Error(0)
}

func (m *MockOrganizationRepository) WithTx(_ bun.IDB) repositories.OrganizationRepository {
	return m
}

type MockOrganizationMemberRepository struct {
	mock.Mock
	Hooks repositories.OrganizationMemberHookExecutor
	byID  map[string]*types.OrganizationMember
}

func (m *MockOrganizationMemberRepository) SetOrganizationMemberHooks(hooks repositories.OrganizationMemberHookExecutor) {
	m.Hooks = hooks
}

func (m *MockOrganizationMemberRepository) Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeCreateOrganizationMember(member); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, member)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	created := args.Get(0).(*types.OrganizationMember)
	if m.Hooks != nil {
		if err := m.Hooks.AfterCreateOrganizationMember(*created); err != nil {
			return nil, err
		}
	}
	return created, args.Error(1)
}

func (m *MockOrganizationMemberRepository) CountByOrganizationID(ctx context.Context, organizationID string) (int, error) {
	args := m.Called(ctx, organizationID)
	return args.Int(0), args.Error(1)
}

func (m *MockOrganizationMemberRepository) GetByOrganizationIDAndUserID(ctx context.Context, organizationID, userID string) (*types.OrganizationMember, error) {
	for _, expectedCall := range m.ExpectedCalls {
		if expectedCall.Method == "GetByOrganizationIDAndUserID" {
			args := m.Called(ctx, organizationID, userID)
			if args.Get(0) == nil {
				return nil, args.Error(1)
			}
			return args.Get(0).(*types.OrganizationMember), args.Error(1)
		}
	}
	return nil, nil
}

func (m *MockOrganizationMemberRepository) GetAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	args := m.Called(ctx, organizationID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberRepository) GetAllByUserID(ctx context.Context, userID string) ([]types.OrganizationMember, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberRepository) GetByID(ctx context.Context, memberID string) (*types.OrganizationMember, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	member := args.Get(0).(*types.OrganizationMember)
	if m.byID == nil {
		m.byID = map[string]*types.OrganizationMember{}
	}
	m.byID[memberID] = member
	return member, args.Error(1)
}

func (m *MockOrganizationMemberRepository) Update(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeUpdateOrganizationMember(member); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, member)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	updated := args.Get(0).(*types.OrganizationMember)
	if m.Hooks != nil {
		if err := m.Hooks.AfterUpdateOrganizationMember(*updated); err != nil {
			return nil, err
		}
	}
	return updated, args.Error(1)
}

func (m *MockOrganizationMemberRepository) Delete(ctx context.Context, memberID string) error {
	if m.Hooks != nil {
		member := m.byID[memberID]
		if member == nil {
			var err error
			member, err = m.GetByID(ctx, memberID)
			if err != nil {
				return err
			}
		}
		if member == nil {
			return nil
		}
		if err := m.Hooks.BeforeDeleteOrganizationMember(member); err != nil {
			return err
		}
		if err := m.Called(ctx, memberID).Error(0); err != nil {
			return err
		}
		return m.Hooks.AfterDeleteOrganizationMember(*member)
	}
	return m.Called(ctx, memberID).Error(0)
}

func (m *MockOrganizationMemberRepository) WithTx(_ bun.IDB) repositories.OrganizationMemberRepository {
	return m
}

type MockOrganizationInvitationRepository struct {
	mock.Mock
	Hooks repositories.OrganizationInvitationHookExecutor
}

func (m *MockOrganizationInvitationRepository) SetOrganizationInvitationHooks(hooks repositories.OrganizationInvitationHookExecutor) {
	m.Hooks = hooks
}

func (m *MockOrganizationInvitationRepository) Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeCreateOrganizationInvitation(invitation); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, invitation)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	created := args.Get(0).(*types.OrganizationInvitation)
	if m.Hooks != nil {
		if err := m.Hooks.AfterCreateOrganizationInvitation(*created); err != nil {
			return nil, err
		}
	}
	return created, args.Error(1)
}

func (m *MockOrganizationInvitationRepository) GetByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationRepository) GetByOrganizationIDAndEmail(ctx context.Context, organizationID, email string, status ...types.OrganizationInvitationStatus) (*types.OrganizationInvitation, error) {
	callArgs := []any{ctx, organizationID, email}
	for _, invitationStatus := range status {
		callArgs = append(callArgs, invitationStatus)
	}
	args := m.Called(callArgs...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationRepository) GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationInvitation, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationRepository) GetAllPendingByEmail(ctx context.Context, email string) ([]types.OrganizationInvitation, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationRepository) Update(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeUpdateOrganizationInvitation(invitation); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, invitation)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	updated := args.Get(0).(*types.OrganizationInvitation)
	if m.Hooks != nil {
		if err := m.Hooks.AfterUpdateOrganizationInvitation(*updated); err != nil {
			return nil, err
		}
	}
	return updated, args.Error(1)
}

func (m *MockOrganizationInvitationRepository) CountByOrganizationIDAndEmail(ctx context.Context, organizationID, email string) (int, error) {
	args := m.Called(ctx, organizationID, email)
	return args.Int(0), args.Error(1)
}

func (m *MockOrganizationInvitationRepository) WithTx(_ bun.IDB) repositories.OrganizationInvitationRepository {
	return m
}

type MockOrganizationTeamRepository struct {
	mock.Mock
	Hooks repositories.OrganizationTeamHookExecutor
	byID  map[string]*types.OrganizationTeam
}

func (m *MockOrganizationTeamRepository) SetOrganizationTeamHooks(hooks repositories.OrganizationTeamHookExecutor) {
	m.Hooks = hooks
}

func (m *MockOrganizationTeamRepository) Create(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeCreateOrganizationTeam(team); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, team)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	created := args.Get(0).(*types.OrganizationTeam)
	if m.Hooks != nil {
		if err := m.Hooks.AfterCreateOrganizationTeam(*created); err != nil {
			return nil, err
		}
	}
	return created, args.Error(1)
}

func (m *MockOrganizationTeamRepository) GetByID(ctx context.Context, teamID string) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	team := args.Get(0).(*types.OrganizationTeam)
	if m.byID == nil {
		m.byID = map[string]*types.OrganizationTeam{}
	}
	m.byID[teamID] = team
	return team, args.Error(1)
}

func (m *MockOrganizationTeamRepository) GetByOrganizationIDAndSlug(ctx context.Context, organizationID, slug string) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, organizationID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamRepository) GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationTeam, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamRepository) Update(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeUpdateOrganizationTeam(team); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, team)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	updated := args.Get(0).(*types.OrganizationTeam)
	if m.Hooks != nil {
		if err := m.Hooks.AfterUpdateOrganizationTeam(*updated); err != nil {
			return nil, err
		}
	}
	return updated, args.Error(1)
}

func (m *MockOrganizationTeamRepository) Delete(ctx context.Context, teamID string) error {
	if m.Hooks != nil {
		team := m.byID[teamID]
		if team == nil {
			var err error
			team, err = m.GetByID(ctx, teamID)
			if err != nil {
				return err
			}
		}
		if team == nil {
			return nil
		}
		if err := m.Hooks.BeforeDeleteOrganizationTeam(team); err != nil {
			return err
		}
		if err := m.Called(ctx, teamID).Error(0); err != nil {
			return err
		}
		return m.Hooks.AfterDeleteOrganizationTeam(*team)
	}
	return m.Called(ctx, teamID).Error(0)
}

func (m *MockOrganizationTeamRepository) WithTx(_ bun.IDB) repositories.OrganizationTeamRepository {
	return m
}

type MockOrganizationTeamMemberRepository struct {
	mock.Mock
	Hooks repositories.OrganizationTeamMemberHookExecutor
	byKey map[string]*types.OrganizationTeamMember
}

func (m *MockOrganizationTeamMemberRepository) SetOrganizationTeamMemberHooks(hooks repositories.OrganizationTeamMemberHookExecutor) {
	m.Hooks = hooks
}

func (m *MockOrganizationTeamMemberRepository) Create(ctx context.Context, teamMember *types.OrganizationTeamMember) (*types.OrganizationTeamMember, error) {
	if m.Hooks != nil {
		if err := m.Hooks.BeforeCreateOrganizationTeamMember(teamMember); err != nil {
			return nil, err
		}
	}
	args := m.Called(ctx, teamMember)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	created := args.Get(0).(*types.OrganizationTeamMember)
	if m.Hooks != nil {
		if err := m.Hooks.AfterCreateOrganizationTeamMember(*created); err != nil {
			return nil, err
		}
	}
	return created, args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetByID(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, teamMemberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, teamID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	teamMember := args.Get(0).(*types.OrganizationTeamMember)
	if m.byKey == nil {
		m.byKey = map[string]*types.OrganizationTeamMember{}
	}
	m.byKey[teamID+":"+memberID] = teamMember
	return teamMember, args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetAllByTeamID(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	args := m.Called(ctx, teamID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) DeleteByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) error {
	if m.Hooks != nil {
		key := teamID + ":" + memberID
		teamMember := m.byKey[key]
		if teamMember == nil {
			var err error
			teamMember, err = m.GetByTeamIDAndMemberID(ctx, teamID, memberID)
			if err != nil {
				return err
			}
		}
		if teamMember == nil {
			return nil
		}
		if err := m.Hooks.BeforeDeleteOrganizationTeamMember(teamMember); err != nil {
			return err
		}
		if err := m.Called(ctx, teamID, memberID).Error(0); err != nil {
			return err
		}
		return m.Hooks.AfterDeleteOrganizationTeamMember(*teamMember)
	}
	return m.Called(ctx, teamID, memberID).Error(0)
}

func (m *MockOrganizationTeamMemberRepository) WithTx(_ bun.IDB) repositories.OrganizationTeamMemberRepository {
	return m
}
