package openapiexport

import (
	"errors"

	internalopenapi "github.com/Authula/authula/internal/openapi"
	"github.com/Authula/authula/openapi"

	accesscontrolopenapi "github.com/Authula/authula/plugins/access-control/openapi"
	adminopenapi "github.com/Authula/authula/plugins/admin/openapi"
	apikeyopenapi "github.com/Authula/authula/plugins/api-key/openapi"
	emailpasswordopenapi "github.com/Authula/authula/plugins/email-password/openapi"
	jwtopenapi "github.com/Authula/authula/plugins/jwt/openapi"
	magiclinkopenapi "github.com/Authula/authula/plugins/magic-link/openapi"
	oauth2openapi "github.com/Authula/authula/plugins/oauth2/openapi"
	organizationsopenapi "github.com/Authula/authula/plugins/organizations/openapi"
	totpopenapi "github.com/Authula/authula/plugins/totp/openapi"
)

func RegisterAllOpenAPIDocs(svc openapi.OpenAPIService, extra ...OpenAPIDocFunc) error {
	var errs []error

	errs = append(errs, internalopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, emailpasswordopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, oauth2openapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, magiclinkopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, jwtopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, totpopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, organizationsopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, accesscontrolopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, adminopenapi.RegisterOpenAPIDocs(svc))
	errs = append(errs, apikeyopenapi.RegisterOpenAPIDocs(svc))

	for _, fn := range extra {
		errs = append(errs, fn(svc))
	}

	return errors.Join(errs...)
}
