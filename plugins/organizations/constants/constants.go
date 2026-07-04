package constants

// Permissions

const (
	All                                       = "organizations:*"
	OrganizationsCreatePermission             = "organizations:create"
	OrganizationsListPermission               = "organizations:list"
	OrganizationsReadPermission               = "organizations:read"
	OrganizationsUpdatePermission             = "organizations:update"
	OrganizationsDeletePermission             = "organizations:delete"
	OrganizationsMembersAddPermission         = "organizations:members:add"
	OrganizationsMembersListPermission        = "organizations:members:list"
	OrganizationsMembersReadPermission        = "organizations:members:read"
	OrganizationsMembersUpdatePermission      = "organizations:members:update"
	OrganizationsMembersRemovePermission      = "organizations:members:remove"
	OrganizationsTeamsCreatePermission        = "organizations:teams:create"
	OrganizationsTeamsListPermission          = "organizations:teams:list"
	OrganizationsTeamsReadPermission          = "organizations:teams:read"
	OrganizationsTeamsUpdatePermission        = "organizations:teams:update"
	OrganizationsTeamsDeletePermission        = "organizations:teams:delete"
	OrganizationsTeamMembersAddPermission     = "organizations:team-members:add"
	OrganizationsTeamMembersListPermission    = "organizations:team-members:list"
	OrganizationsTeamMembersReadPermission    = "organizations:team-members:read"
	OrganizationsTeamMembersRemovePermission  = "organizations:team-members:remove"
	OrganizationsInvitationsCreatePermission  = "organizations:invitations:create"
	OrganizationsInvitationsListPermission    = "organizations:invitations:list"
	OrganizationsInvitationsReadPermission    = "organizations:invitations:read"
	OrganizationsInvitationsRevokePermission  = "organizations:invitations:revoke"
	OrganizationsInvitationsProcessPermission = "organizations:invitations:process"
)

// Events

const (
	EventOrganizationsInvitationCreated = "organizations.invitation.created"
)
