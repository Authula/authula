package services

type JWTService interface {
	ValidateToken(token string) (userID string, err error)
}
