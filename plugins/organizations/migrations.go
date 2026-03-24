package organizations

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/organizations/migrationset"
)

func organizationsMigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.ForProvider(provider)
}
