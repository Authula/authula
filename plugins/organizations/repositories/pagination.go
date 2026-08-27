package repositories

import (
	"github.com/Authula/authula/core/pagination"
)

// pageLimit guards only against a non-positive limit producing a LIMIT 0 query.
// There is deliberately no ceiling here: the maximum page size is configurable per
// deployment and lives in the service layer (ServiceUtils.ClampPagination), which is
// the only layer that can see the plugin config. Capping to a constant here would
// silently override an operator who raised max_page_limit. Every HTTP path reaches
// these repositories through a service that has already clamped; a direct caller is
// a Go embedder, on the same footing as the unpaginated GetAll methods.
func pageLimit(limit int) int {
	if limit <= 0 {
		return pagination.DefaultLimit
	}
	return limit
}

func pageOffset(page, limit int) int {
	if page < 1 || limit <= 0 {
		return 0
	}
	return (page - 1) * limit
}
