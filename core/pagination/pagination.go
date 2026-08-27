package pagination

import (
	"net/http"

	"github.com/Authula/authula/util"
)

const (
	DefaultPage  = 1
	DefaultLimit = 10
	// DefaultMaxLimit is the fallback ceiling on a page size. It exists to stop a
	// caller-supplied limit from turning a paginated query into a full table scan.
	DefaultMaxLimit = 100
)

type Params struct {
	Page  int
	Limit int
}

type Pagination struct {
	Page       int  `json:"page" required:"true" nullable:"false"`
	Limit      int  `json:"limit" required:"true" nullable:"false"`
	Total      int  `json:"total" required:"true" nullable:"false"`
	TotalPages int  `json:"total_pages" required:"true" nullable:"false"`
	HasMore    bool `json:"has_more" required:"true" nullable:"false"`
}

// Clamp coerces params into a range that is safe to hand to a database: the page
// starts at 1, and the limit is held between 1 and maxLimit. A maxLimit of zero or
// less falls back to DefaultMaxLimit, so a caller that has nothing configured still
// gets a bounded query rather than an unbounded one.
//
// The floor is applied before the ceiling, so a maxLimit below DefaultLimit also
// caps the default: with maxLimit 5, an unset limit yields 5 rather than 10.
func Clamp(params Params, maxLimit int) Params {
	if maxLimit <= 0 {
		maxLimit = DefaultMaxLimit
	}
	if params.Page < DefaultPage {
		params.Page = DefaultPage
	}
	if params.Limit <= 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit > maxLimit {
		params.Limit = maxLimit
	}
	return params
}

func New(page, limit, total int) Pagination {
	if limit <= 0 {
		limit = DefaultLimit
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	return Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

func ParseFromRequest(r *http.Request) Params {
	return Params{
		Page:  util.GetQueryInt(r, "page", DefaultPage),
		Limit: util.GetQueryInt(r, "limit", DefaultLimit),
	}
}
