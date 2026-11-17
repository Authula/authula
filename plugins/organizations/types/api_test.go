package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	coretypes "github.com/Authula/authula/core/types"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestRequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validate  func() error
		expectErr error
		verify    func(*testing.T)
	}{
		{
			name: "create organization trims name, slug and logo",
			validate: func() error {
				request := &types.CreateOrganizationRequest{Name: "  Acme  ", Role: "owner", Slug: new("  acme  "), Logo: new("  logo.png  ")}
				err := request.Validate()
				require.Equal(t, "  Acme  ", request.Name)
				require.Equal(t, "acme", *request.Slug)
				require.Equal(t, "logo.png", *request.Logo)
				return err
			},
		},
		{
			name: "create organization with blank name is unprocessable",
			validate: func() error {
				return (&types.CreateOrganizationRequest{Name: "   ", Role: "owner"}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "create organization with blank role is unprocessable",
			validate: func() error {
				return (&types.CreateOrganizationRequest{Name: "Acme", Role: " "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "create organization leaves nil optional fields alone",
			validate: func() error {
				request := &types.CreateOrganizationRequest{Name: "Acme", Role: "owner"}
				err := request.Validate()
				require.Nil(t, request.Slug)
				require.Nil(t, request.Logo)
				return err
			},
		},
		{
			name: "update organization trims name, slug and logo",
			validate: func() error {
				request := &types.UpdateOrganizationRequest{
					Name: new("  Acme  "),
					Slug: new("  acme  "),
					Logo: coretypes.Optional[*string]{Value: new("  logo.png  "), Present: true},
				}
				err := request.Validate()
				require.Equal(t, "Acme", *request.Name)
				require.Equal(t, "acme", *request.Slug)
				require.Equal(t, "logo.png", *request.Logo.Value)
				return err
			},
		},
		{
			name: "update organization rejects a blank name",
			validate: func() error {
				return (&types.UpdateOrganizationRequest{Name: new("   ")}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "update organization tolerates a present nil logo",
			validate: func() error {
				request := &types.UpdateOrganizationRequest{Logo: coretypes.Optional[*string]{Present: true}}
				return request.Validate()
			},
		},
		{
			name: "update organization leaves absent optional fields alone",
			validate: func() error {
				request := &types.UpdateOrganizationRequest{}
				err := request.Validate()
				require.Nil(t, request.Name)
				require.Nil(t, request.Slug)
				require.False(t, request.Logo.Present)
				return err
			},
		},
		{
			name: "create invitation accepts valid input",
			validate: func() error {
				return (&types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"}).Validate()
			},
		},
		{
			name: "create invitation rejects a blank email",
			validate: func() error {
				return (&types.CreateOrganizationInvitationRequest{Email: "  ", Role: "member"}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "create invitation rejects a blank role",
			validate: func() error {
				return (&types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: " "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "add member accepts valid input",
			validate: func() error {
				return (&types.AddOrganizationMemberRequest{UserID: "user-1", Role: "member"}).Validate()
			},
		},
		{
			name: "add member rejects a blank user",
			validate: func() error {
				return (&types.AddOrganizationMemberRequest{UserID: " ", Role: "member"}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "add member rejects a blank role",
			validate: func() error {
				return (&types.AddOrganizationMemberRequest{UserID: "user-1", Role: " "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "update member accepts a role",
			validate: func() error {
				return (&types.UpdateOrganizationMemberRequest{Role: "admin"}).Validate()
			},
		},
		{
			name: "update member rejects a blank role",
			validate: func() error {
				return (&types.UpdateOrganizationMemberRequest{Role: "  "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "create team trims slug and description",
			validate: func() error {
				request := &types.CreateOrganizationTeamRequest{Name: "Platform", Slug: new("  platform  "), Description: new("  core team  ")}
				err := request.Validate()
				require.Equal(t, "platform", *request.Slug)
				require.Equal(t, "core team", *request.Description)
				return err
			},
		},
		{
			name: "create team rejects a blank name",
			validate: func() error {
				return (&types.CreateOrganizationTeamRequest{Name: "  "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "create team leaves nil optional fields alone",
			validate: func() error {
				request := &types.CreateOrganizationTeamRequest{Name: "Platform"}
				err := request.Validate()
				require.Nil(t, request.Slug)
				require.Nil(t, request.Description)
				return err
			},
		},
		{
			name: "update team trims slug and description",
			validate: func() error {
				request := &types.UpdateOrganizationTeamRequest{Name: "Platform", Slug: new("  platform  "), Description: new("  core team  ")}
				err := request.Validate()
				require.Equal(t, "platform", *request.Slug)
				require.Equal(t, "core team", *request.Description)
				return err
			},
		},
		{
			name: "update team rejects a blank name",
			validate: func() error {
				return (&types.UpdateOrganizationTeamRequest{Name: " "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "update team leaves nil optional fields alone",
			validate: func() error {
				request := &types.UpdateOrganizationTeamRequest{Name: "Platform"}
				err := request.Validate()
				require.Nil(t, request.Slug)
				require.Nil(t, request.Description)
				return err
			},
		},
		{
			name: "add team member accepts a member id",
			validate: func() error {
				return (&types.AddOrganizationTeamMemberRequest{MemberID: "mem-1"}).Validate()
			},
		},
		{
			name: "add team member rejects a blank member id",
			validate: func() error {
				return (&types.AddOrganizationTeamMemberRequest{MemberID: " "}).Validate()
			},
			expectErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "accept invitation trims the redirect url",
			validate: func() error {
				request := &types.AcceptOrganizationInvitationRequest{RedirectURL: new("  https://app.example.com  ")}
				err := request.Validate()
				require.Equal(t, "https://app.example.com", *request.RedirectURL)
				return err
			},
		},
		{
			name: "accept invitation tolerates a nil redirect url",
			validate: func() error {
				request := &types.AcceptOrganizationInvitationRequest{}
				err := request.Validate()
				require.Nil(t, request.RedirectURL)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.validate()
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func roundTrip[T any](t *testing.T, input T) T {
	t.Helper()

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var output T
	require.NoError(t, json.Unmarshal(data, &output))
	return output
}

func TestRequestAndResponseJSONRoundTrips(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create organization request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.CreateOrganizationRequest{Name: "Acme", Role: "owner", Slug: new("acme"), Logo: new("logo.png"), Metadata: map[string]any{"plan": "pro"}})
				require.Equal(t, "Acme", got.Name)
				require.Equal(t, "owner", got.Role)
				require.Equal(t, "acme", *got.Slug)
				require.Equal(t, "logo.png", *got.Logo)
				require.Equal(t, "pro", got.Metadata["plan"])
			},
		},
		{
			name: "update organization request omits an absent logo",
			run: func(t *testing.T) {
				t.Helper()
				data, err := json.Marshal(types.UpdateOrganizationRequest{Name: new("Acme")})
				require.NoError(t, err)

				var raw map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(data, &raw))
				require.NotContains(t, raw, "logo")

				var got types.UpdateOrganizationRequest
				require.NoError(t, json.Unmarshal(data, &got))
				require.Equal(t, "Acme", *got.Name)
				require.False(t, got.Logo.Present)
			},
		},
		{
			name: "update organization request carries a present logo",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.UpdateOrganizationRequest{Logo: coretypes.Optional[*string]{Value: new("logo.png"), Present: true}})
				require.True(t, got.Logo.Present)
				require.Equal(t, "logo.png", *got.Logo.Value)
			},
		},
		{
			name: "create invitation request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"})
				require.Equal(t, "bob@example.com", got.Email)
				require.Equal(t, "member", got.Role)
			},
		},
		{
			name: "add member request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.AddOrganizationMemberRequest{UserID: "user-1", Role: "admin"})
				require.Equal(t, "user-1", got.UserID)
				require.Equal(t, "admin", got.Role)
			},
		},
		{
			name: "update member request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.UpdateOrganizationMemberRequest{Role: "admin"})
				require.Equal(t, "admin", got.Role)
			},
		},
		{
			name: "create team request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.CreateOrganizationTeamRequest{Name: "Platform", Slug: new("platform"), Description: new("core"), Metadata: map[string]any{"k": "v"}})
				require.Equal(t, "Platform", got.Name)
				require.Equal(t, "platform", *got.Slug)
				require.Equal(t, "core", *got.Description)
				require.Equal(t, "v", got.Metadata["k"])
			},
		},
		{
			name: "update team request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.UpdateOrganizationTeamRequest{Name: "Platform", Slug: new("platform"), Description: new("core")})
				require.Equal(t, "Platform", got.Name)
				require.Equal(t, "platform", *got.Slug)
				require.Equal(t, "core", *got.Description)
			},
		},
		{
			name: "add team member request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.AddOrganizationTeamMemberRequest{MemberID: "mem-1"})
				require.Equal(t, "mem-1", got.MemberID)
			},
		},
		{
			name: "accept invitation request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.AcceptOrganizationInvitationRequest{RedirectURL: new("https://app.example.com")})
				require.Equal(t, "https://app.example.com", *got.RedirectURL)
			},
		},
		{
			name: "list organizations request",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.ListOrganizationsRequest{Page: 2, Limit: 25})
				require.Equal(t, 2, got.Page)
				require.Equal(t, 25, got.Limit)
			},
		},
		{
			name: "list response with pagination",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.ListOrganizationsResponse{
					Data:       []types.Organization{{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"}},
					Pagination: pagination.New(1, 10, 1),
				})
				require.Len(t, got.Data, 1)
				require.Equal(t, "org-1", got.Data[0].ID)
				require.Equal(t, 1, got.Pagination.Total)
			},
		},
		{
			name: "invitation with organization summary",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.GetOrganizationInvitationResponse{
					Invitation:   &types.OrganizationInvitation{ID: "inv-1", Email: "bob@example.com", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: now},
					Organization: types.OrganizationSummary{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme", Logo: new("logo.png"), Metadata: map[string]any{"plan": "pro"}},
				})
				require.NotNil(t, got.Invitation)
				require.Equal(t, types.OrganizationInvitationStatusPending, got.Invitation.Status)
				require.Equal(t, "Acme", got.Organization.Name)
				require.Equal(t, "pro", got.Organization.Metadata["plan"])
			},
		},
		{
			name: "member response",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.OrganizationMemberResponse{
					ID:             "mem-1",
					OrganizationID: "org-1",
					Role:           "member",
					CreatedAt:      now,
					UpdatedAt:      now,
					User:           models.User{ID: "user-1", Name: "Bob", Email: "bob@example.com", EmailVerified: true},
				})
				require.Equal(t, "mem-1", got.ID)
				require.Equal(t, now, got.CreatedAt)
				require.Equal(t, "bob@example.com", got.User.Email)
				require.True(t, got.User.EmailVerified)
			},
		},
		{
			name: "team member response",
			run: func(t *testing.T) {
				t.Helper()
				got := roundTrip(t, types.OrganizationTeamMemberResponse{
					ID:        "tm-1",
					TeamID:    "team-1",
					CreatedAt: now,
					Member:    types.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1", Role: "member", User: models.User{ID: "user-1"}},
				})
				require.Equal(t, "tm-1", got.ID)
				require.Equal(t, "team-1", got.TeamID)
				require.Equal(t, "mem-1", got.Member.ID)
			},
		},
		{
			name: "delete responses",
			run: func(t *testing.T) {
				t.Helper()
				require.Equal(t, "deleted", roundTrip(t, types.DeleteOrganizationResponse{Message: "deleted"}).Message)
				require.Equal(t, "deleted", roundTrip(t, types.DeleteOrganizationMemberResponse{Message: "deleted"}).Message)
				require.Equal(t, "deleted", roundTrip(t, types.DeleteOrganizationTeamResponse{Message: "deleted"}).Message)
				require.Equal(t, "deleted", roundTrip(t, types.DeleteOrganizationTeamMemberResponse{Message: "deleted"}).Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.run(t)
		})
	}
}
