package constants

import "strings"

// Permissions

const (
	OrganizationsAllPermission               = "organizations:*"
	OrganizationsReadPermission              = "organizations:read"
	OrganizationsUpdatePermission            = "organizations:update"
	OrganizationsDeletePermission            = "organizations:delete"
	OrganizationsMembersAddPermission        = "organizations:members:add"
	OrganizationsMembersListPermission       = "organizations:members:list"
	OrganizationsMembersReadPermission       = "organizations:members:read"
	OrganizationsMembersUpdatePermission     = "organizations:members:update"
	OrganizationsMembersRemovePermission     = "organizations:members:remove"
	OrganizationsTeamsCreatePermission       = "organizations:teams:create"
	OrganizationsTeamsListPermission         = "organizations:teams:list"
	OrganizationsTeamsReadPermission         = "organizations:teams:read"
	OrganizationsTeamsUpdatePermission       = "organizations:teams:update"
	OrganizationsTeamsDeletePermission       = "organizations:teams:delete"
	OrganizationsTeamMembersAddPermission    = "organizations:team-members:add"
	OrganizationsTeamMembersListPermission   = "organizations:team-members:list"
	OrganizationsTeamMembersReadPermission   = "organizations:team-members:read"
	OrganizationsTeamMembersRemovePermission = "organizations:team-members:remove"
	OrganizationsInvitationsCreatePermission = "organizations:invitations:create"
	OrganizationsInvitationsListPermission   = "organizations:invitations:list"
	OrganizationsInvitationsReadPermission   = "organizations:invitations:read"
	OrganizationsInvitationsRevokePermission = "organizations:invitations:revoke"
)

// OrganizationPermissions is the full set of permissions the organizations plugin registers.
var OrganizationPermissions = []string{
	OrganizationsAllPermission,
	OrganizationsReadPermission,
	OrganizationsUpdatePermission,
	OrganizationsDeletePermission,
	OrganizationsMembersAddPermission,
	OrganizationsMembersListPermission,
	OrganizationsMembersReadPermission,
	OrganizationsMembersUpdatePermission,
	OrganizationsMembersRemovePermission,
	OrganizationsTeamsCreatePermission,
	OrganizationsTeamsListPermission,
	OrganizationsTeamsReadPermission,
	OrganizationsTeamsUpdatePermission,
	OrganizationsTeamsDeletePermission,
	OrganizationsTeamMembersAddPermission,
	OrganizationsTeamMembersListPermission,
	OrganizationsTeamMembersReadPermission,
	OrganizationsTeamMembersRemovePermission,
	OrganizationsInvitationsCreatePermission,
	OrganizationsInvitationsListPermission,
	OrganizationsInvitationsReadPermission,
	OrganizationsInvitationsRevokePermission,
}

// CoversAllOrganizationPermissions reports whether the given permission set grants
// every organization permission (wildcard-aware).
func CoversAllOrganizationPermissions(permissions []string) bool {
	for _, required := range OrganizationPermissions {
		if !matchesPermission(permissions, required) {
			return false
		}
	}
	return true
}

func matchesPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == required {
			return true
		}
		if strings.HasSuffix(permission, "*") && strings.HasPrefix(required, strings.TrimSuffix(permission, "*")) {
			return true
		}
	}
	return false
}

// Events

const (
	EventOrganizationsInvitationCreated = "organizations.invitation.created"
)
