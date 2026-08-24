package organizations

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/handlers"
)

func Routes(plugin *OrganizationsPlugin) []models.Route {
	createOrganizationHandler := &handlers.CreateOrganizationHandler{UseCases: plugin.useCases}
	getAllOrganizationsHandler := &handlers.GetAllOrganizationsHandler{UseCases: plugin.useCases}
	getOrganizationByIDHandler := &handlers.GetOrganizationByIDHandler{UseCases: plugin.useCases}
	updateOrganizationHandler := &handlers.UpdateOrganizationHandler{UseCases: plugin.useCases}
	deleteOrganizationHandler := &handlers.DeleteOrganizationHandler{UseCases: plugin.useCases}

	createInvitationHandler := &handlers.CreateOrganizationInvitationHandler{UseCases: plugin.useCases}
	getInvitationHandler := &handlers.GetOrganizationInvitationHandler{UseCases: plugin.useCases}
	getAllInvitationsHandler := &handlers.GetAllOrganizationInvitationsHandler{UseCases: plugin.useCases}
	revokeInvitationHandler := &handlers.RevokeOrganizationInvitationHandler{UseCases: plugin.useCases}
	acceptInvitationHandler := &handlers.AcceptOrganizationInvitationHandler{UseCases: plugin.useCases, TrustedOrigins: plugin.globalConfig.Security.TrustedOrigins}
	rejectInvitationHandler := &handlers.RejectOrganizationInvitationHandler{UseCases: plugin.useCases}

	addMemberHandler := &handlers.AddOrganizationMemberHandler{UseCases: plugin.useCases}
	getAllMembersHandler := &handlers.GetAllOrganizationMembersHandler{UseCases: plugin.useCases}
	getMemberHandler := &handlers.GetOrganizationMemberHandler{UseCases: plugin.useCases}
	getMemberByUserIDHandler := &handlers.GetOrganizationMemberByUserIDHandler{UseCases: plugin.useCases}
	updateMemberHandler := &handlers.UpdateOrganizationMemberHandler{UseCases: plugin.useCases}
	deleteMemberHandler := &handlers.DeleteOrganizationMemberHandler{UseCases: plugin.useCases}

	createTeamHandler := &handlers.CreateOrganizationTeamHandler{UseCases: plugin.useCases}
	getAllTeamsHandler := &handlers.GetAllOrganizationTeamsHandler{UseCases: plugin.useCases}
	getTeamHandler := &handlers.GetOrganizationTeamHandler{UseCases: plugin.useCases}
	updateTeamHandler := &handlers.UpdateOrganizationTeamHandler{UseCases: plugin.useCases}
	deleteTeamHandler := &handlers.DeleteOrganizationTeamHandler{UseCases: plugin.useCases}

	addTeamMemberHandler := &handlers.AddOrganizationTeamMemberHandler{UseCases: plugin.useCases}
	getAllTeamMembersHandler := &handlers.GetAllOrganizationTeamMembersHandler{UseCases: plugin.useCases}
	getTeamMemberHandler := &handlers.GetOrganizationTeamMemberHandler{UseCases: plugin.useCases}
	deleteTeamMemberHandler := &handlers.DeleteOrganizationTeamMemberHandler{UseCases: plugin.useCases}

	return []models.Route{
		// Organizations
		{
			Method: http.MethodPost,
			Path:   "/organizations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
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
				middleware.RequireActor(models.ActorUser),
			},
			Handler: updateOrganizationHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: deleteOrganizationHandler.Handle(),
		},
		// Invitations
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/invitations",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
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
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/revoke",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: revokeInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/accept",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: acceptInvitationHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/invitations/{invitation_id}/reject",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: rejectInvitationHandler.Handle(),
		},
		// Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
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
			Method: http.MethodGet,
			Path:   "/organizations/{organization_id}/members/by-user/{user_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireAuthenticated(),
			},
			Handler: getMemberByUserIDHandler.Handle(),
		},
		{
			Method: http.MethodPatch,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: updateMemberHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/members/{member_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: deleteMemberHandler.Handle(),
		},
		// Teams
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
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
				middleware.RequireActor(models.ActorUser),
			},
			Handler: updateTeamHandler.Handle(),
		},
		{
			Method: http.MethodDelete,
			Path:   "/organizations/{organization_id}/teams/{team_id}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: deleteTeamHandler.Handle(),
		},
		// Team Members
		{
			Method: http.MethodPost,
			Path:   "/organizations/{organization_id}/teams/{team_id}/members",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
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
				middleware.RequireActor(models.ActorUser),
			},
			Handler: deleteTeamMemberHandler.Handle(),
		},
	}
}
