package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"

	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationInvitationStatus_PrepareJSONSchema(t *testing.T) {
	t.Parallel()

	var schema jsonschema.Schema
	require.NoError(t, types.OrganizationInvitationStatus("").PrepareJSONSchema(&schema))

	require.NotNil(t, schema.Type)
	require.NotNil(t, schema.Type.SimpleTypes)
	require.Equal(t, jsonschema.String, *schema.Type.SimpleTypes)
	require.Empty(t, schema.Type.SliceOfSimpleTypeValues)
	require.Equal(t, []any{
		"pending",
		"accepted",
		"rejected",
		"revoked",
		"expired",
	}, schema.Enum)
	require.NotNil(t, schema.Description)
	require.Equal(t, "The type of the invitation status", *schema.Description)
}

func TestInvitationStatusConstantsAreStable(t *testing.T) {
	t.Parallel()

	require.Equal(t, types.OrganizationInvitationStatus("pending"), types.OrganizationInvitationStatusPending)
	require.Equal(t, types.OrganizationInvitationStatus("accepted"), types.OrganizationInvitationStatusAccepted)
	require.Equal(t, types.OrganizationInvitationStatus("rejected"), types.OrganizationInvitationStatusRejected)
	require.Equal(t, types.OrganizationInvitationStatus("revoked"), types.OrganizationInvitationStatusRevoked)
	require.Equal(t, types.OrganizationInvitationStatus("expired"), types.OrganizationInvitationStatusExpired)
}
