package usecases

type UseCases struct {
	AuthorizeUseCase   *AuthorizeUseCase
	CallbackUseCase    *CallbackUseCase
	RefreshUseCase     *RefreshUseCase
	LinkAccountUseCase *LinkAccountUseCase
}
