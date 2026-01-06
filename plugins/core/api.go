package core

import "github.com/GoBetterAuth/go-better-auth/plugins/core/usecases"

type API struct {
	useCases *usecases.UseCases
}

func BuildAPI(plugin *CorePlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

func BuildUseCases(p *CorePlugin) *usecases.UseCases {
	return &usecases.UseCases{
		HealthCheckUseCase: &usecases.HealthCheckUseCase{},
		GetMeUseCase: &usecases.GetMeUseCase{
			Logger:      p.logger,
			UserService: p.userService,
		},
		SignOutUseCase: &usecases.SignOutUseCase{
			Logger:         p.logger,
			SessionService: p.sessionService,
		},
	}
}
