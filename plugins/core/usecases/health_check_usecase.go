package usecases

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/plugins/core/types"
)

type HealthCheckUseCase struct{}

func (uc *HealthCheckUseCase) HealthCheck(ctx context.Context) (*types.HealthCheckResult, error) {
	return &types.HealthCheckResult{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}, nil
}
