package me

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type MeUseCase interface {
	GetMe(ctx context.Context, userID string) (*models.MeResult, error)
}
