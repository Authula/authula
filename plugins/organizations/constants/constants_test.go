package constants_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/organizations/constants"
)

func TestCoversAllOrganizationPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		permissions []string
		expected    bool
	}{
		{
			name:        "nil grants nothing",
			permissions: nil,
			expected:    false,
		},
		{
			name:        "empty grants nothing",
			permissions: []string{},
			expected:    false,
		},
		{
			name:        "global wildcard covers everything",
			permissions: []string{"*"},
			expected:    true,
		},
		{
			name:        "organization wildcard covers everything",
			permissions: []string{"organizations:*"},
			expected:    true,
		},
		{
			name:        "every permission listed explicitly covers everything",
			permissions: constants.OrganizationPermissions,
			expected:    true,
		},
		{
			name:        "a wildcard list covering each group covers everything",
			permissions: []string{"organizations:*", "organizations:members:*", "organizations:teams:*", "organizations:team-members:*", "organizations:invitations:*"},
			expected:    true,
		},
		{
			name:        "one missing explicit permission fails",
			permissions: explicitPermissions()[:len(explicitPermissions())-1],
			expected:    false,
		},
		{
			name:        "unrelated permissions fail",
			permissions: []string{"users:read", "sessions:*"},
			expected:    false,
		},
		{
			name: "a narrower wildcard does not cover a sibling",
			permissions: []string{
				"organizations:members:*",
				"organizations:teams:*",
				"organizations:invitations:*",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, constants.CoversAllOrganizationPermissions(tt.permissions))
		})
	}
}

// explicitPermissions returns the registered permissions with the organizations:*
// wildcard removed, so a single missing scope is detectable.
func explicitPermissions() []string {
	permissions := make([]string, 0, len(constants.OrganizationPermissions))
	for _, permission := range constants.OrganizationPermissions {
		if permission == constants.OrganizationsAllPermission {
			continue
		}
		permissions = append(permissions, permission)
	}
	return permissions
}

func TestOrganizationPermissionsAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, len(constants.OrganizationPermissions))
	for _, permission := range constants.OrganizationPermissions {
		_, exists := seen[permission]
		require.False(t, exists, "duplicate permission %q", permission)
		seen[permission] = struct{}{}
	}
}

func TestTemplateEnvironmentKeysAreStable(t *testing.T) {
	t.Parallel()

	require.Equal(t, "AUTHULA_TEMPLATE_ORGANIZATION_INVITATION_SUBJECT", constants.EnvOrganizationInvitationSubject)
	require.Equal(t, "AUTHULA_TEMPLATE_ORGANIZATION_INVITATION_TEXT", constants.EnvOrganizationInvitationText)
	require.Equal(t, "AUTHULA_TEMPLATE_ORGANIZATION_INVITATION_HTML", constants.EnvOrganizationInvitationHTML)
	require.Equal(t, "organizations.invitation.created", constants.EventOrganizationsInvitationCreated)
}
