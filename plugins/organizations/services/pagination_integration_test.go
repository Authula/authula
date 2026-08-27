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

func TestOrganizationMemberService_ListAllMembersEnforcesLimitsAgainstSQL(t *testing.T) {
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
			name:             "a large limit is capped at the maximum page size",
			params:           pagination.Params{Page: 1, Limit: 100000},
			expectLen:        memberCount,
			expectPagination: pagination.Pagination{Page: 1, Limit: pagination.DefaultMaxLimit, Total: memberCount, TotalPages: 1, HasMore: false},
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

			resp, err := svc.ListAllMembers(ctx, orgtests.Actor("user-1"), "org-1", tt.params)
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
	resp, err := svc.ListAllOrganizations(ctx, orgtests.Actor("user-1"), pagination.Params{Page: 1, Limit: 1})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1, "a single-row page")
	require.Equal(t, limit, resp.Pagination.Total, "the total must not be capped by the page size")

	var seen []types.Organization
	for page := 1; page <= limit; page++ {
		pageResp, err := svc.ListAllOrganizations(ctx, orgtests.Actor("user-1"), pagination.Params{Page: page, Limit: 1})
		require.NoError(t, err)
		seen = append(seen, pageResp.Data...)
	}
	require.Len(t, seen, limit, "paging must yield every accessible organization exactly once")
}

// The unconstrained sibling of ListAllMembers must return the whole collection in
// one call, against real SQL, where the paginated path needs two pages.
func TestOrganizationMemberService_GetAllMembersReturnsEverythingInOneCallAgainstSQL(t *testing.T) {
	t.Parallel()

	const memberCount = 12

	db := orgtests.SetupRepoDB(t)
	orgtests.SeedUsers(t, db, memberCount)
	orgtests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	orgtests.SeedOrganization(t, db, "org-2", "user-1", "Beta Inc", "beta-inc")
	for i := 1; i <= memberCount; i++ {
		orgtests.SeedOrganizationMember(t, db, fmt.Sprintf("mem-%02d", i), "org-1", fmt.Sprintf("user-%d", i), "member")
	}
	orgtests.SeedOrganizationMember(t, db, "mem-other", "org-2", "user-1", "owner")

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
	ctx := context.Background()
	actor := orgtests.Actor("user-1")

	// The paginated path only reaches the whole set by walking pages.
	page, err := svc.ListAllMembers(ctx, actor, "org-1", pagination.Params{})
	require.NoError(t, err)
	require.Len(t, page.Data, pagination.DefaultLimit)
	require.True(t, page.Pagination.HasMore, "the default page must not cover the whole collection")

	members, err := svc.GetAllMembers(ctx, actor, "org-1")
	require.NoError(t, err)
	require.Len(t, members, memberCount, "one call must return every member")

	// Members of another organization must not leak in, and each row is hydrated.
	for _, member := range members {
		require.Equal(t, "org-1", member.OrganizationID)
		require.NotEmpty(t, member.User.ID)
		require.NotEmpty(t, member.User.Email)
	}
}

// The ceiling on a page size must survive all the way into the SQL LIMIT, and the
// configured value — not the constant — must be the one that wins.
func TestOrganizationMemberService_ListAllMembersRespectsTheConfiguredMaxPageLimitAgainstSQL(t *testing.T) {
	t.Parallel()

	const memberCount = 120

	setup := func(t *testing.T, maxPageLimit int) (*organizationMemberService, context.Context) {
		t.Helper()

		db := orgtests.SetupRepoDB(t)
		orgtests.SeedUsers(t, db, memberCount)
		orgtests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		for i := 1; i <= memberCount; i++ {
			orgtests.SeedOrganizationMember(t, db, fmt.Sprintf("mem-%03d", i), "org-1", fmt.Sprintf("user-%d", i), "member")
		}

		orgRepo := repositories.NewBunOrganizationRepository(db)
		memberRepo := repositories.NewBunOrganizationMemberRepository(db)
		serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo, maxPageLimit: maxPageLimit}
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
		maxPageLimit     int
		params           pagination.Params
		expectLen        int
		expectPagination pagination.Pagination
	}{
		{
			name:             "an unconfigured maximum truncates the page at the default maximum",
			maxPageLimit:     0,
			params:           pagination.Params{Page: 1, Limit: 1_000_000},
			expectLen:        pagination.DefaultMaxLimit,
			expectPagination: pagination.Pagination{Page: 1, Limit: 100, Total: memberCount, TotalPages: 2, HasMore: true},
		},
		{
			name:             "a configured maximum below the default maximum wins",
			maxPageLimit:     5,
			params:           pagination.Params{Page: 1, Limit: 1_000_000},
			expectLen:        5,
			expectPagination: pagination.Pagination{Page: 1, Limit: 5, Total: memberCount, TotalPages: 24, HasMore: true},
		},
		{
			name:             "a configured maximum above the default maximum wins",
			maxPageLimit:     120,
			params:           pagination.Params{Page: 1, Limit: 1_000_000},
			expectLen:        memberCount,
			expectPagination: pagination.Pagination{Page: 1, Limit: 120, Total: memberCount, TotalPages: 1, HasMore: false},
		},
		{
			name:             "a limit within the configured maximum is untouched",
			maxPageLimit:     120,
			params:           pagination.Params{Page: 2, Limit: 30},
			expectLen:        30,
			expectPagination: pagination.Pagination{Page: 2, Limit: 30, Total: memberCount, TotalPages: 4, HasMore: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx := setup(t, tt.maxPageLimit)

			resp, err := svc.ListAllMembers(ctx, orgtests.Actor("user-1"), "org-1", tt.params)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Data, tt.expectLen)
			require.Equal(t, tt.expectPagination, resp.Pagination)
		})
	}
}

// The unpaginated sibling is the sanctioned escape hatch for whole collections and
// must stay uncapped, however low the configured page-size ceiling is.
func TestOrganizationMemberService_GetAllMembersIgnoresTheMaxPageLimitAgainstSQL(t *testing.T) {
	t.Parallel()

	const memberCount = 120

	db := orgtests.SetupRepoDB(t)
	orgtests.SeedUsers(t, db, memberCount)
	orgtests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	for i := 1; i <= memberCount; i++ {
		orgtests.SeedOrganizationMember(t, db, fmt.Sprintf("mem-%03d", i), "org-1", fmt.Sprintf("user-%d", i), "member")
	}

	orgRepo := repositories.NewBunOrganizationRepository(db)
	memberRepo := repositories.NewBunOrganizationMemberRepository(db)
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo, maxPageLimit: 5}
	svc := NewOrganizationMemberService(
		&internaltests.MockUserService{},
		orgtests.NewAccessControlServiceStub(),
		orgRepo,
		memberRepo,
		nil,
		&orgtests.MockTxRunner{},
		serviceUtils,
	)

	members, err := svc.GetAllMembers(context.Background(), orgtests.Actor("user-1"), "org-1")
	require.NoError(t, err)
	require.Len(t, members, memberCount, "a max page limit of 5 must not truncate the unpaginated call")
}
