package accesscontrol

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/access-control/migrationset"
)

func accessControlMigrations() []migrations.Migration {
	return migrationset.Migrations()
}
