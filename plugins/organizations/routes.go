package organizations

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/handlers"
)

func Routes(plugin *OrganizationsPlugin) []models.Route {
	createOrganizationHandler := &handlers.CreateOrganizationHandler{OrgService: plugin.organizationService}
	getAllOrganizationsHandler := &handlers.GetAllOrganizationsHandler{OrgService: plugin.organizationService}
	getOrganizationByIDHandler := &handlers.GetOrganizationByIDHandler{OrgService: plugin.organizationService}
	updateOrganizationHandler := &handlers.UpdateOrganizationHandler{OrgService: plugin.organizationService}
	deleteOrganizationHandler := &handlers.DeleteOrganizationHandler{OrgService: plugin.organizationService}

	createInvitationHandler := &handlers.CreateOrganizationInvitationHandler{OrgInvitationService: plugin.invitationService}
	getInvitationHandler := &handlers.GetOrganizationInvitationHandler{OrgInvitationService: plugin.invitationService}
	getAllInvitationsHandler := &handlers.GetAllOrganizationInvitationsHandler{OrgInvitationService: plugin.invitationService}
	revokeInvitationHandler := &handlers.RevokeOrganizationInvitationHandler{OrgInvitationService: plugin.invitationService}
	acceptInvitationHandler := &handlers.AcceptOrganizationInvitationHandler{OrgInvitationService: plugin.invitationService}
	rejectInvitationHandler := &handlers.RejectOrganizationInvitationHandler{OrgInvitationService: plugin.invitationService}

	addMemberHandler := &handlers.AddOrganizationMemberHandler{OrgMemberService: plugin.memberService}
	getAllMembersHandler := &handlers.GetAllOrganizationMembersHandler{OrgMemberService: plugin.memberService}
	getMemberHandler := &handlers.GetOrganizationMemberHandler{OrgMemberService: plugin.memberService}
	updateMemberHandler := &handlers.UpdateOrganizationMemberHandler{OrgMemberService: plugin.memberService}
	deleteMemberHandler := &handlers.DeleteOrganizationMemberHandler{OrgMemberService: plugin.memberService}

	createTeamHandler := &handlers.CreateOrganizationTeamHandler{OrgTeamService: plugin.teamService}
	getAllTeamsHandler := &handlers.GetAllOrganizationTeamsHandler{OrgTeamService: plugin.teamService}
	getTeamHandler := &handlers.GetOrganizationTeamHandler{OrgTeamService: plugin.teamService}
	updateTeamHandler := &handlers.UpdateOrganizationTeamHandler{OrgTeamService: plugin.teamService}
	deleteTeamHandler := &handlers.DeleteOrganizationTeamHandler{OrgTeamService: plugin.teamService}

	addTeamMemberHandler := &handlers.AddOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.teamMemberService}
	getAllTeamMembersHandler := &handlers.GetAllOrganizationTeamMembersHandler{OrgTeamMemberService: plugin.teamMemberService}
	getTeamMemberHandler := &handlers.GetOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.teamMemberService}
	deleteTeamMemberHandler := &handlers.DeleteOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.teamMemberService}

	return []models.Route{
		// Organizations
		{
			Method: http.MethodPost,
			Path:   "/organizations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: createOrganizationHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getAllOrganizationsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getOrganizationByIDHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: updateOrganizationHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: deleteOrganizationHandler.Handle(),
		},
		// Invitations
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: createInvitationHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/invitations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getAllInvitationsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: revokeInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/accept",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: acceptInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/reject",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: rejectInvitationHandler.Handle(),
		},
		// Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: addMemberHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getAllMembersHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getMemberHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: updateMemberHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: deleteMemberHandler.Handle(),
		},
		// Teams
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: createTeamHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getAllTeamsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getTeamHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: updateTeamHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: deleteTeamHandler.Handle(),
		},
		// Team Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: addTeamMemberHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getAllTeamMembersHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: getTeamMemberHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser, models.ActorMachine),
			},
			Handler: deleteTeamMemberHandler.Handle(),
		},
	}
}
