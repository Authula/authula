package apikey

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/api-key/migrationset"
)

func apiKeyMigrations() []migrations.Migration {
	return migrationset.Migrations()
}
