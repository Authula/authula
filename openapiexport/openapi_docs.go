package openapiexport

import (
	"errors"

	"github.com/Authula/authula/openapi"
	emailpasswordopenapi "github.com/Authula/authula/plugins/email-password/openapi"
	oauth2openapi "github.com/Authula/authula/plugins/oauth2/openapi"
)

func RegisterAllOpenAPIDocs(svc openapi.OpenAPIService, basePath string, extra ...OpenAPIDocFunc) error {
	var errs []error

	errs = append(errs, emailpasswordopenapi.RegisterOpenAPIDocs(svc, basePath))
	errs = append(errs, oauth2openapi.RegisterOpenAPIDocs(svc, basePath))

	for _, fn := range extra {
		errs = append(errs, fn(svc, basePath))
	}

	return errors.Join(errs...)
}
