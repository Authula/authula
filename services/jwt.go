package services

import (
	"context"

	"github.com/Authula/authula/models"
)

type JWTService interface {
	ValidateToken(ctx context.Context, token string) (*models.Actor, error)
}
