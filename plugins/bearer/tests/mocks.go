package tests

import (
	internaltests "github.com/Authula/authula/internal/tests"
	rootservices "github.com/Authula/authula/services"
)

type MockJWTService = internaltests.MockJWTService

var _ rootservices.JWTService = (*MockJWTService)(nil)
