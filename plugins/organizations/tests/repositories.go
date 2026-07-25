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

func (m *MockOrganizationRepository) Create(ctx context.Context, organization *types.Organization) (*types.Organization, error) {
	args := m.Called(ctx, organization)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetByID(ctx context.Context, organizationID string) (*types.Organization, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
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
	args := m.Called(ctx, organization)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, organizationID string) error {
	return m.Called(ctx, organizationID).Error(0)
}

func (m *MockOrganizationRepository) WithTx(_ bun.IDB) repositories.OrganizationRepository {
	return m
}

type MockOrganizationMemberRepository struct {
	mock.Mock
}

func (m *MockOrganizationMemberRepository) Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	args := m.Called(ctx, member)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
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

func (m *MockOrganizationMemberRepository) GetAllByOrganizationIDWithUser(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, organizationID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationMemberResponse), args.Error(1)
}

func (m *MockOrganizationMemberRepository) GetByIDWithUser(ctx context.Context, memberID string) (*types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMemberResponse), args.Error(1)
}

func (m *MockOrganizationMemberRepository) GetByOrganizationIDAndUserIDWithUser(ctx context.Context, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMemberResponse), args.Error(1)
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
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberRepository) Update(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	args := m.Called(ctx, member)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberRepository) Delete(ctx context.Context, memberID string) error {
	return m.Called(ctx, memberID).Error(0)
}

func (m *MockOrganizationMemberRepository) WithTx(_ bun.IDB) repositories.OrganizationMemberRepository {
	return m
}

type MockOrganizationInvitationRepository struct {
	mock.Mock
}

func (m *MockOrganizationInvitationRepository) Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, invitation)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
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
	args := m.Called(ctx, invitation)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
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
}

func (m *MockOrganizationTeamRepository) Create(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, team)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamRepository) GetByID(ctx context.Context, teamID string) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
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
	args := m.Called(ctx, team)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamRepository) Delete(ctx context.Context, teamID string) error {
	return m.Called(ctx, teamID).Error(0)
}

func (m *MockOrganizationTeamRepository) WithTx(_ bun.IDB) repositories.OrganizationTeamRepository {
	return m
}

type MockOrganizationTeamMemberRepository struct {
	mock.Mock
}

func (m *MockOrganizationTeamMemberRepository) Create(ctx context.Context, teamMember *types.OrganizationTeamMember) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, teamMember)
	if mockResultIsNil(args.Get(0)) {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
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
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetAllByTeamID(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	args := m.Called(ctx, teamID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetAllByTeamIDWithMemberAndUser(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMemberResponse, error) {
	args := m.Called(ctx, teamID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeamMemberResponse), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) GetByIDWithMemberAndUser(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMemberResponse, error) {
	args := m.Called(ctx, teamMemberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMemberResponse), args.Error(1)
}

func (m *MockOrganizationTeamMemberRepository) DeleteByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) error {
	return m.Called(ctx, teamID, memberID).Error(0)
}

func (m *MockOrganizationTeamMemberRepository) WithTx(_ bun.IDB) repositories.OrganizationTeamMemberRepository {
	return m
}
