package ratelimit

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/rate-limit/migrationset"
)

func rateLimitMigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.ForProvider(provider)
}
