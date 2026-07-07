package organizations

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/organizations/migrationset"
)

func organizationsMigrations() []migrations.Migration {
	return migrationset.Migrations()
}
