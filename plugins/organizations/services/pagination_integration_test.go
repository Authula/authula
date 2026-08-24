package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/organizations/repositories"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationMemberService_GetAllMembersEnforcesLimitsAgainstSQL(t *testing.T) {
	t.Parallel()

	const memberCount = 12

	setup := func(t *testing.T) (*organizationMemberService, context.Context) {
		t.Helper()

		db := orgtests.SetupRepoDB(t)
		orgtests.SeedUsers(t, db, memberCount)
		orgtests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		for i := 1; i <= memberCount; i++ {
			orgtests.SeedOrganizationMember(t, db, fmt.Sprintf("mem-%02d", i), "org-1", fmt.Sprintf("user-%d", i), "member")
		}

		orgRepo := repositories.NewBunOrganizationRepository(db)
		memberRepo := repositories.NewBunOrganizationMemberRepository(db)
		serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
		svc := NewOrganizationMemberService(
			&internaltests.MockUserService{},
			orgtests.NewAccessControlServiceStub(),
			orgRepo,
			memberRepo,
			nil,
			&orgtests.MockTxRunner{},
			serviceUtils,
		)

		return svc, context.Background()
	}

	tests := []struct {
		name             string
		params           pagination.Params
		expectLen        int
		expectPagination pagination.Pagination
	}{
		{
			name:             "no params applies the defaults",
			params:           pagination.Params{},
			expectLen:        pagination.DefaultLimit,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: memberCount, TotalPages: 2, HasMore: true},
		},
		{
			name:             "a large limit is honoured",
			params:           pagination.Params{Page: 1, Limit: 100000},
			expectLen:        memberCount,
			expectPagination: pagination.Pagination{Page: 1, Limit: 100000, Total: memberCount, TotalPages: 1, HasMore: false},
		},
		{
			name:             "a negative limit does not read the whole table",
			params:           pagination.Params{Page: 1, Limit: -1},
			expectLen:        pagination.DefaultLimit,
			expectPagination: pagination.Pagination{Page: 1, Limit: 10, Total: memberCount, TotalPages: 2, HasMore: true},
		},
		{
			name:             "page zero returns the first page",
			params:           pagination.Params{Page: 0, Limit: 5},
			expectLen:        5,
			expectPagination: pagination.Pagination{Page: 1, Limit: 5, Total: memberCount, TotalPages: 3, HasMore: true},
		},
		{
			name:             "a page past the end is empty with a correct total",
			params:           pagination.Params{Page: 99999, Limit: 10},
			expectLen:        0,
			expectPagination: pagination.Pagination{Page: 99999, Limit: 10, Total: memberCount, TotalPages: 2, HasMore: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx := setup(t)

			resp, err := svc.GetAllMembers(ctx, orgtests.Actor("user-1"), "org-1", tt.params)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Data, tt.expectLen)
			require.Equal(t, tt.expectPagination, resp.Pagination)
		})
	}
}

func TestOrganizationService_QuotaSurvivesPagination(t *testing.T) {
	t.Parallel()

	const limit = 3

	db := orgtests.SetupRepoDB(t)
	orgRepo := repositories.NewBunOrganizationRepository(db)
	memberRepo := repositories.NewBunOrganizationMemberRepository(db)
	ctx := context.Background()

	// user-1 owns one organization and is a member of two more it does not own.
	orgtests.SeedOrganization(t, db, "org-owned", "user-1", "Owned", "owned")
	orgtests.SeedOrganizationMember(t, db, "mem-owned", "org-owned", "user-1", "owner")
	orgtests.SeedOrganization(t, db, "org-joined-1", "user-2", "Joined One", "joined-one")
	orgtests.SeedOrganizationMember(t, db, "mem-joined-1", "org-joined-1", "user-1", "member")

	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo}
	svc := NewOrganizationService(orgRepo, memberRepo, serviceUtils, nil, new(limit), nil)

	require.NoError(t, svc.ensureOrganizationLimit(ctx, orgtests.Actor("user-1"), orgRepo), "two organizations is below the quota")

	orgtests.SeedOrganization(t, db, "org-joined-2", "user-2", "Joined Two", "joined-two")
	orgtests.SeedOrganizationMember(t, db, "mem-joined-2", "org-joined-2", "user-1", "member")

	require.Error(t, svc.ensureOrganizationLimit(ctx, orgtests.Actor("user-1"), orgRepo), "the quota must reject at the boundary")

	// The quota counts the same set the list endpoint returns, deduped.
	resp, err := svc.GetAllOrganizations(ctx, orgtests.Actor("user-1"), pagination.Params{Page: 1, Limit: 1})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1, "a single-row page")
	require.Equal(t, limit, resp.Pagination.Total, "the total must not be capped by the page size")

	var seen []types.Organization
	for page := 1; page <= limit; page++ {
		pageResp, err := svc.GetAllOrganizations(ctx, orgtests.Actor("user-1"), pagination.Params{Page: page, Limit: 1})
		require.NoError(t, err)
		seen = append(seen, pageResp.Data...)
	}
	require.Len(t, seen, limit, "paging must yield every accessible organization exactly once")
}
