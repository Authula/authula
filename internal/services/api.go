package services

type Api struct {
	Users         UserService
	Accounts      AccountService
	Sessions      SessionService
	Verifications VerificationService
	Tokens        TokenService
	// TODO: KeyValueStore *services.KeyValueStoreService
}
