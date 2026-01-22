package usecases

type UseCases struct {
	SignUpUseCase                *SignUpUseCase
	SignInUseCase                *SignInUseCase
	VerifyEmailUseCase           *VerifyEmailUseCase
	SendEmailVerificationUseCase *SendEmailVerificationUseCase
	RequestPasswordResetUseCase  *RequestPasswordResetUseCase
	ChangePasswordUseCase        *ChangePasswordUseCase
	RequestEmailChangeUseCase    *RequestEmailChangeUseCase
}
