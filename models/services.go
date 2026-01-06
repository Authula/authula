package models

type ServiceID string

const (
	// CORE
	ServiceUser         ServiceID = "user_service"
	ServiceAccount      ServiceID = "account_service"
	ServiceSession      ServiceID = "session_service"
	ServiceVerification ServiceID = "verification_service"
	ServiceToken        ServiceID = "token_service"

	// EMAIL
	ServicePassword ServiceID = "password_service"
	ServiceMailer   ServiceID = "mailer_service"

	// JWT
	ServiceJWT ServiceID = "jwt_service"

	// CONFIG
	ServiceConfigManager ServiceID = "config_manager_service"

	// STORAGE
	ServiceSecondaryStorage ServiceID = "secondary_storage_service"
)

func (id ServiceID) String() string {
	return string(id)
}

type ServiceRegistry interface {
	Register(name string, service any)
	Get(name string) any
}
