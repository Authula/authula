package models

type ServiceID string

const (
	// CORE
	ServiceUser         ServiceID = "user_service"
	ServiceAccount      ServiceID = "account_service"
	ServiceSession      ServiceID = "session_service"
	ServiceVerification ServiceID = "verification_service"
	ServiceToken        ServiceID = "token_service"
	ServicePassword     ServiceID = "password_service"

	// Plugins
	ServiceAccessControl    ServiceID = "access_control_service"
	ServiceAdmin            ServiceID = "admin_service"
	ServiceSecondaryStorage ServiceID = "secondary_storage_service"
	ServiceMailer           ServiceID = "mailer_service"
	ServiceJWT              ServiceID = "jwt_service"
	ServiceConfigManager    ServiceID = "config_manager_service"
)

func (id ServiceID) String() string {
	return string(id)
}

type ServiceRegistry interface {
	Register(name string, service any)
	Get(name string) any
}
