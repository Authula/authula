package accesscontrol

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/access-control/migrationset"
)

func accessControlMigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.ForProvider(provider)
}
