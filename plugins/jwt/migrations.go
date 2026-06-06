package jwt

import (
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/plugins/jwt/migrationset"
)

func JWTMigrationsForProvider(provider string) []migrations.Migration {
	return migrationset.JWTMigrationsForProvider(provider)
}
