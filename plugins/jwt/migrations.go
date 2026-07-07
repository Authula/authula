package jwt

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/jwt/migrationset"
)

func JWTMigrations() []migrations.Migration {
	return migrationset.Migrations()
}
