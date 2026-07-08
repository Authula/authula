package openapiexport

import (
	"errors"

	"github.com/Authula/authula/openapi"
	emailpasswordopenapi "github.com/Authula/authula/plugins/email-password/openapi"
	jwtopenapi "github.com/Authula/authula/plugins/jwt/openapi"
	magiclinkopenapi "github.com/Authula/authula/plugins/magic-link/openapi"
	oauth2openapi "github.com/Authula/authula/plugins/oauth2/openapi"
	organizationsopenapi "github.com/Authula/authula/plugins/organizations/openapi"
	totpopenapi "github.com/Authula/authula/plugins/totp/openapi"
)

func RegisterAllOpenAPIDocs(svc openapi.OpenAPIService, basePath string, extra ...OpenAPIDocFunc) error {
	var errs []error

	errs = append(errs, emailpasswordopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, oauth2openapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, magiclinkopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, jwtopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, totpopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, organizationsopenapi.RegisterOpenAPIDocs(svc, basePath))

	for _, fn := range extra {
		errs = append(errs, fn(svc, basePath))
	}

	return errors.Join(errs...)
}
