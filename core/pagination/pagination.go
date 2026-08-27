package pagination

import (
	"net/http"

	"github.com/Authula/authula/util"
)

const (
	DefaultPage     = 1
	DefaultLimit    = 10
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
