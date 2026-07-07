package ratelimit

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/rate-limit/migrationset"
)

func ratelimitMigrations() []migrations.Migration {
	return migrationset.Migrations()
}
