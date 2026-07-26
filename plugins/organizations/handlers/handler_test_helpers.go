package handlers

import (
	"context"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgusecases "github.com/Authula/authula/plugins/organizations/usecases"
)

type noopAuthorizer struct{}

func (a *noopAuthorizer) AuthorizeScope(_ context.Context, _ *models.Actor, _ string) error {
	return nil
}

func (a *noopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *models.Actor, _ string) error {
	return nil
}

func defaultMockUserService() *internaltests.MockUserService {
	svc := &internaltests.MockUserService{}
	svc.On("GetByID", mock.Anything, mock.Anything).Return((*models.User)(nil), nil).Maybe()
	svc.On("GetByEmail", mock.Anything, mock.Anything).Return((*models.User)(nil), nil).Maybe()
	return svc
}

func newOrgUseCases(svc orgservices.OrganizationService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		svc,
		&orgtests.MockOrganizationInvitationService{},
		&orgtests.MockOrganizationMemberService{},
		&orgtests.MockOrganizationTeamService{},
		&orgtests.MockOrganizationTeamMemberService{},
		defaultMockUserService(),
		&models.Config{},
		&noopAuthorizer{},
	)
}

func newInvitationUseCases(orgSvc orgservices.OrganizationService, svc orgservices.OrganizationInvitationService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		orgSvc,
		svc,
		&orgtests.MockOrganizationMemberService{},
		&orgtests.MockOrganizationTeamService{},
		&orgtests.MockOrganizationTeamMemberService{},
		defaultMockUserService(),
		&models.Config{},
		&noopAuthorizer{},
	)
}

func newMemberUseCases(svc orgservices.OrganizationMemberService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		&orgtests.MockOrganizationService{},
		&orgtests.MockOrganizationInvitationService{},
		svc,
		&orgtests.MockOrganizationTeamService{},
		&orgtests.MockOrganizationTeamMemberService{},
		defaultMockUserService(),
		&models.Config{},
		&noopAuthorizer{},
	)
}

func newTeamUseCases(svc orgservices.OrganizationTeamService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		&orgtests.MockOrganizationService{},
		&orgtests.MockOrganizationInvitationService{},
		&orgtests.MockOrganizationMemberService{},
		svc,
		&orgtests.MockOrganizationTeamMemberService{},
		defaultMockUserService(),
		&models.Config{},
		&noopAuthorizer{},
	)
}

func newTeamMemberUseCases(svc orgservices.OrganizationTeamMemberService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		&orgtests.MockOrganizationService{},
		&orgtests.MockOrganizationInvitationService{},
		&orgtests.MockOrganizationMemberService{},
		&orgtests.MockOrganizationTeamService{},
		svc,
		defaultMockUserService(),
		&models.Config{},
		&noopAuthorizer{},
	)
}
