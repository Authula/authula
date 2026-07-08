package openapiexport

import (
	"github.com/Authula/authula/openapi"
	emailpasswordopenapi "github.com/Authula/authula/plugins/email-password/openapi"
	oauth2openapi "github.com/Authula/authula/plugins/oauth2/openapi"
)

func RegisterAllOpenAPIDocs(svc openapi.OpenAPIService, basePath string, extra ...OpenAPIDocFunc) {
	emailpasswordopenapi.RegisterOpenAPIDocs(svc, basePath)
	oauth2openapi.RegisterOpenAPIDocs(svc, basePath)

	for _, fn := range extra {
		fn(svc, basePath)
	}
}
