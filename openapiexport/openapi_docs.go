package openapiexport

import (
	"errors"

	accesscontrolopenapi "github.com/Authula/authula/plugins/access-control/openapi"
	adminopenapi "github.com/Authula/authula/plugins/admin/openapi"
	apikeyopenapi "github.com/Authula/authula/plugins/api-key/openapi"

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
	errs = append(errs, accesscontrolopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, adminopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, apikeyopenapi.RegisterOpenAPIDocs(svc, basePath))

	for _, fn := range extra {
		errs = append(errs, fn(svc, basePath))
	}

	return errors.Join(errs...)
}
