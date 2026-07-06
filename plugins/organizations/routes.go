package organizations

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/handlers"
)

func Routes(plugin *OrganizationsPlugin) []models.Route {
	createOrganizationHandler := &handlers.CreateOrganizationHandler{OrgService: plugin.useCases}
	getAllOrganizationsHandler := &handlers.GetAllOrganizationsHandler{OrgService: plugin.useCases}
	getOrganizationByIDHandler := &handlers.GetOrganizationByIDHandler{OrgService: plugin.useCases}
	updateOrganizationHandler := &handlers.UpdateOrganizationHandler{OrgService: plugin.useCases}
	deleteOrganizationHandler := &handlers.DeleteOrganizationHandler{OrgService: plugin.useCases}

	createInvitationHandler := &handlers.CreateOrganizationInvitationHandler{OrgInvitationService: plugin.useCases}
	getInvitationHandler := &handlers.GetOrganizationInvitationHandler{OrgInvitationService: plugin.useCases}
	getAllInvitationsHandler := &handlers.GetAllOrganizationInvitationsHandler{OrgInvitationService: plugin.useCases}
	revokeInvitationHandler := &handlers.RevokeOrganizationInvitationHandler{OrgInvitationService: plugin.useCases}
	acceptInvitationHandler := &handlers.AcceptOrganizationInvitationHandler{OrgInvitationService: plugin.useCases}
	rejectInvitationHandler := &handlers.RejectOrganizationInvitationHandler{OrgInvitationService: plugin.useCases}

	addMemberHandler := &handlers.AddOrganizationMemberHandler{OrgMemberService: plugin.useCases}
	getAllMembersHandler := &handlers.GetAllOrganizationMembersHandler{OrgMemberService: plugin.useCases}
	getMemberHandler := &handlers.GetOrganizationMemberHandler{OrgMemberService: plugin.useCases}
	updateMemberHandler := &handlers.UpdateOrganizationMemberHandler{OrgMemberService: plugin.useCases}
	deleteMemberHandler := &handlers.DeleteOrganizationMemberHandler{OrgMemberService: plugin.useCases}

	createTeamHandler := &handlers.CreateOrganizationTeamHandler{OrgTeamService: plugin.useCases}
	getAllTeamsHandler := &handlers.GetAllOrganizationTeamsHandler{OrgTeamService: plugin.useCases}
	getTeamHandler := &handlers.GetOrganizationTeamHandler{OrgTeamService: plugin.useCases}
	updateTeamHandler := &handlers.UpdateOrganizationTeamHandler{OrgTeamService: plugin.useCases}
	deleteTeamHandler := &handlers.DeleteOrganizationTeamHandler{OrgTeamService: plugin.useCases}

	addTeamMemberHandler := &handlers.AddOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.useCases}
	getAllTeamMembersHandler := &handlers.GetAllOrganizationTeamMembersHandler{OrgTeamMemberService: plugin.useCases}
	getTeamMemberHandler := &handlers.GetOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.useCases}
	deleteTeamMemberHandler := &handlers.DeleteOrganizationTeamMemberHandler{OrgTeamMemberService: plugin.useCases}

	return []models.Route{
		// Organizations
		{
			Method: http.MethodPost,
			Path:   "/organizations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: createOrganizationHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllOrganizationsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getOrganizationByIDHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: updateOrganizationHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: deleteOrganizationHandler.Handle(),
		},
		// Invitations
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: createInvitationHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/invitations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllInvitationsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: revokeInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/accept",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: acceptInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/reject",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: rejectInvitationHandler.Handle(),
		},
		// Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: addMemberHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllMembersHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getMemberHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: updateMemberHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: deleteMemberHandler.Handle(),
		},
		// Teams
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: createTeamHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllTeamsHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getTeamHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: updateTeamHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: deleteTeamHandler.Handle(),
		},
		// Team Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: addTeamMemberHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getAllTeamMembersHandler.Handle(),
		},
		{
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getTeamMemberHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: deleteTeamMemberHandler.Handle(),
		},
	}
}
