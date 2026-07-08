package openapiexport

import (
	"fmt"
	"os"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/openapi"
)

type OpenAPIDocFunc func(svc openapi.OpenAPIService, basePath string) error

func ExportSpecToFile(outputPath, format string, extraDocs []OpenAPIDocFunc) error {
	return ExportSpecToFileWithVersion(outputPath, format, "3.1.0", extraDocs)
}

func ExportSpecToFileWithVersion(outputPath, format, openAPIVersion string, extraDocs []OpenAPIDocFunc) error {
	if format != "json" && format != "yaml" {
		return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
	}

	env.LoadEnvConfig()

	svc, err := GenerateService(
		"Authula API",
		env.GetEnv(env.EnvOpenAPISpecVersion, "0.1.0"),
		"Authula API - An open-source authentication solution that scales with you.",
		env.GetEnv(env.EnvBaseURL, "http://localhost:8080"),
		"/api/auth",
		extraDocs,
		openapi.WithOpenAPIVersion(openAPIVersion),
		openapi.WithShortSchemaNames(),
	)
	if err != nil {
		return fmt.Errorf("initializing OpenAPI service: %w", err)
	}

	var data []byte
	switch format {
	case "yaml":
		data, err = svc.SpecYAML()
	case "json":
		data, err = svc.SpecJSON()
	}
	if err != nil {
		return fmt.Errorf("marshalling OpenAPI spec: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing OpenAPI spec: %w", err)
	}

	return nil
}

func GenerateService(title, apiVersion, description, serverURL, basePath string, extraDocs []OpenAPIDocFunc, opts ...openapi.ServiceOption) (openapi.OpenAPIService, error) {
	service, err := openapi.NewOpenAPIService(title, apiVersion, description, serverURL, opts...)
	if err != nil {
		return nil, err
	}
	if err := RegisterAllOpenAPIDocs(service, basePath, extraDocs...); err != nil {
		return nil, fmt.Errorf("registering OpenAPI docs: %w", err)
	}
	return service, nil
}
