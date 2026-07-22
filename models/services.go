package models

import "context"

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
	ServiceRateLimit        ServiceID = "rate_limit_service"
	ServiceOrganization     ServiceID = "organization_service"
)

func (id ServiceID) String() string {
	return string(id)
}

type ServiceRegistry interface {
	Register(name string, service any)
	Get(name string) any
}

const ContextServiceRegistry ContextKey = "service.service_registry"

func NewContextWithServiceRegistry(ctx context.Context, registry ServiceRegistry) context.Context {
	return context.WithValue(ctx, ContextServiceRegistry, registry)
}

func GetServiceRegistry(ctx context.Context) ServiceRegistry {
	registry, _ := ctx.Value(ContextServiceRegistry).(ServiceRegistry)
	return registry
}

func GetServiceFromContext[T any](ctx context.Context, id ServiceID) (bool, T) {
	registry := GetServiceRegistry(ctx)
	if registry == nil {
		var zero T
		return false, zero
	}
	svc, ok := registry.Get(id.String()).(T)
	return ok, svc
}
