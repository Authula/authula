package apikey

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/api-key/migrationset"
)

func apiKeyMigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.ForProvider(provider)
}
