package handlers

import (
	"context"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
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

func defaultServiceUtils() *orgservices.ServiceUtils {
	orgRepo := &orgtests.MockOrganizationRepository{}
	memberRepo := &orgtests.MockOrganizationMemberRepository{}
	orgRepo.On("GetByID", mock.Anything, mock.Anything).Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Maybe()
	memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, mock.Anything, mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "admin"}, nil).Maybe()
	return orgservices.NewServiceUtils(orgRepo, memberRepo, nil, 0)
}

func defaultAccessControlService() *orgtests.AccessControlServiceStub {
	return orgtests.NewAccessControlServiceStub()
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
		defaultServiceUtils(),
		defaultAccessControlService(),
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
		defaultServiceUtils(),
		defaultAccessControlService(),
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
		defaultServiceUtils(),
		defaultAccessControlService(),
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
		defaultServiceUtils(),
		defaultAccessControlService(),
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
		defaultServiceUtils(),
		defaultAccessControlService(),
	)
}
