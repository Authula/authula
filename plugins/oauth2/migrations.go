package oauth2

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/oauth2/migrationset"
)

func oauth2MigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.ForProvider(provider)
}
