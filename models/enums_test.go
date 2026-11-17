package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"
)

func TestActorTypeToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		actorType ActorType
		want      string
	}{
		{name: "user", actorType: ActorUser, want: "user"},
		{name: "machine", actorType: ActorMachine, want: "machine"},
		{name: "custom", actorType: ActorType("service"), want: "service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.actorType.ToString())
		})
	}
}

func TestAuthProviderIDString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider AuthProviderID
		want     string
	}{
		{name: "email", provider: AuthProviderEmail, want: "email"},
		{name: "magic link", provider: AuthProviderMagicLink, want: "magic-link"},
		{name: "discord", provider: AuthProviderDiscord, want: "discord"},
		{name: "dropbox", provider: AuthProviderDropbox, want: "dropbox"},
		{name: "github", provider: AuthProviderGitHub, want: "github"},
		{name: "google", provider: AuthProviderGoogle, want: "google"},
		{name: "microsoft", provider: AuthProviderMicrosoft, want: "microsoft"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestPluginIDString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pluginID PluginID
		want     string
	}{
		{name: "access control", pluginID: PluginAccessControl, want: "access_control"},
		{name: "email password", pluginID: PluginEmailPassword, want: "email_password"},
		{name: "rate limit", pluginID: PluginRateLimit, want: "ratelimit"},
		{name: "custom", pluginID: PluginID("custom"), want: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.pluginID.String())
		})
	}
}

func TestServiceIDString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		serviceID ServiceID
		want      string
	}{
		{name: "user", serviceID: ServiceUser, want: "user_service"},
		{name: "session", serviceID: ServiceSession, want: "session_service"},
		{name: "jwt", serviceID: ServiceJWT, want: "jwt_service"},
		{name: "custom", serviceID: ServiceID("custom"), want: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.serviceID.String())
		})
	}
}

func TestVerificationTypeString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "email_verification", TypeEmailVerification.String())
	require.Equal(t, "totp_pending_auth", TypeTOTPPendingAuth.String())
	require.Equal(t, "custom", VerificationType("custom").String())
}

func TestVerificationTypePrepareJSONSchema(t *testing.T) {
	t.Parallel()

	schema := &jsonschema.Schema{}

	err := VerificationType("").PrepareJSONSchema(schema)

	require.NoError(t, err)
	require.NotNil(t, schema.Type)
	require.NotNil(t, schema.Type.SimpleTypes)
	require.Equal(t, jsonschema.String, *schema.Type.SimpleTypes)
	require.Equal(t, []any{
		string(TypeEmailVerification),
		string(TypePasswordResetRequest),
		string(TypeEmailResetRequest),
		string(TypeMagicLinkSignInRequest),
		string(TypeMagicLinkExchangeCode),
		string(TypeTOTPPendingAuth),
	}, schema.Enum)
	require.NotNil(t, schema.Description)
	require.Equal(t, "The type of the verification", *schema.Description)
}
