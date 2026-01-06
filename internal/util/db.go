package util

import "github.com/uptrace/bun"

// ApplyFieldUpdates applies map fields to a Bun UpdateQuery safely
func ApplyFieldUpdates(q *bun.UpdateQuery, fields map[string]any) *bun.UpdateQuery {
	for field, value := range fields {
		q = q.Set(field+" = ?", value)
	}
	return q
}
