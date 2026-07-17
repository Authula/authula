package handlers

import (
	"context"

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

func newOrgUseCases(svc orgservices.OrganizationService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		svc,
		&orgtests.MockOrganizationInvitationService{},
		&orgtests.MockOrganizationMemberService{},
		&orgtests.MockOrganizationTeamService{},
		&orgtests.MockOrganizationTeamMemberService{},
		&noopAuthorizer{},
	)
}

func newInvitationUseCases(svc orgservices.OrganizationInvitationService) *orgusecases.UseCases {
	return orgusecases.NewUseCases(
		&orgtests.MockOrganizationService{},
		svc,
		&orgtests.MockOrganizationMemberService{},
		&orgtests.MockOrganizationTeamService{},
		&orgtests.MockOrganizationTeamMemberService{},
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
		&noopAuthorizer{},
	)
}
