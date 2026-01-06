package usecases

type UseCases struct {
	SignUpUseCase                *SignUpUseCase
	SignInUseCase                *SignInUseCase
	VerifyEmailUseCase           *VerifyEmailUseCase
	SendVerificationEmailUseCase *SendVerificationEmailUseCase
	RequestPasswordResetUseCase  *RequestPasswordResetUseCase
	ChangePasswordUseCase        *ChangePasswordUseCase
}
