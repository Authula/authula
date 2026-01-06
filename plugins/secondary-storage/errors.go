package secondarystorage

import "errors"

var (
	ErrRedisConfigURLNotProvided = errors.New("redis secondary storage configuration URL not provided")
	ErrDatabaseConfigNotProvided = errors.New("database secondary storage configuration not provided")
)
