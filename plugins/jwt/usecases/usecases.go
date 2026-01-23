package usecases

type UseCases struct {
	RefreshTokenUseCase RefreshTokenUseCase
	JWKSUseCase         JWKSUseCase
}

func NewUseCases(
	refreshTokenUseCase RefreshTokenUseCase,
	jwksUseCase JWKSUseCase,
) *UseCases {
	return &UseCases{
		RefreshTokenUseCase: refreshTokenUseCase,
		JWKSUseCase:         jwksUseCase,
	}
}
