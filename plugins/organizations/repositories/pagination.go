package repositories

import (
	"github.com/Authula/authula/core/pagination"
)

const MaxPendingInvitationsPerBatch = 500

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
