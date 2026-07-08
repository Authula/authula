package openapi

import (
	"fmt"
	"reflect"

	"github.com/swaggest/jsonschema-go"
	swaggestopenapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/openapi-go/openapi31"
)

type OpenAPIService interface {
	AddOperation(method, pathPattern string, opts ...OperationOption) error
	SpecJSON() ([]byte, error)
	SpecYAML() ([]byte, error)
	RegisterSchema(name string, typ any) error
}

type specMarshaller interface {
	MarshalJSON() ([]byte, error)
	MarshalYAML() ([]byte, error)
}

type openapiService struct {
	reflector swaggestopenapi.Reflector
	specMarshaller
	addToComponents func(name string, sch jsonschema.Schema)
	openAPIVersion  string
	shortNames      bool
}

type ServiceOption func(*openapiService)

// WithOpenAPIVersion sets the OpenAPI spec version. Supported values:
//
//	"3.0.0" – "3.0.3" (default "3.0.3")
//	"3.1.0"
func WithOpenAPIVersion(v string) ServiceOption {
	return func(s *openapiService) {
		s.openAPIVersion = v
	}
}

// WithShortSchemaNames strips Go package prefixes from schema names so that
// a Go type like `types.HealthResponse` becomes `HealthResponse` instead of
// `TypesHealthResponse`. This produces cleaner names when consuming the spec.
func WithShortSchemaNames() ServiceOption {
	return func(s *openapiService) {
		s.shortNames = true
	}
}

func NewOpenAPIService(title, apiVersion, description, serverURL string, opts ...ServiceOption) (OpenAPIService, error) {
	svc := &openapiService{}
	for _, opt := range opts {
		opt(svc)
	}

	if svc.openAPIVersion == "" {
		svc.openAPIVersion = "3.0.3"
	}

	switch svc.openAPIVersion {
	case "3.0.0", "3.0.1", "3.0.2", "3.0.3":
		if err := svc.initV3(title, apiVersion, description, serverURL); err != nil {
			return nil, err
		}

	case "3.1.0":
		if err := svc.initV31(title, apiVersion, description, serverURL); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported OpenAPI version: %s", svc.openAPIVersion)
	}

	return svc, nil
}

func (s *openapiService) initV3(title, apiVersion, description, serverURL string) error {
	r := openapi3.NewReflector()
	r.SpecEns().Info.
		WithTitle(title).
		WithVersion(apiVersion).
		WithDescription(description)
	r.SpecEns().WithServers(openapi3.Server{URL: serverURL})
	r.Spec.Openapi = s.openAPIVersion

	r.JSONSchemaReflector().DefaultOptions = append(
		r.JSONSchemaReflector().DefaultOptions,
		wrapNullableRefInAnyOf,
	)

	if s.shortNames {
		r.JSONSchemaReflector().DefaultOptions = append(
			r.JSONSchemaReflector().DefaultOptions,
			jsonschema.InterceptDefName(func(t reflect.Type, defaultDefName string) string {
				return t.Name()
			}),
		)
	}

	s.reflector = r
	s.specMarshaller = r.Spec
	s.addToComponents = func(name string, sch jsonschema.Schema) {
		schemaOrRef := openapi3.SchemaOrRef{}
		schemaOrRef.FromJSONSchema(sch.ToSchemaOrBool())
		r.SpecEns().ComponentsEns().SchemasEns().
			WithMapOfSchemaOrRefValuesItem(name, schemaOrRef)
	}

	return nil
}

func (s *openapiService) initV31(title, apiVersion, description, serverURL string) error {
	r := openapi31.NewReflector()
	r.SpecEns().Info.
		WithTitle(title).
		WithVersion(apiVersion).
		WithDescription(description)
	r.SpecEns().WithServers(openapi31.Server{URL: serverURL})
	r.Spec.Openapi = s.openAPIVersion

	r.JSONSchemaReflector().DefaultOptions = append(
		r.JSONSchemaReflector().DefaultOptions,
		wrapNullableRefInAnyOf,
	)

	if s.shortNames {
		r.JSONSchemaReflector().DefaultOptions = append(
			r.JSONSchemaReflector().DefaultOptions,
			jsonschema.InterceptDefName(func(t reflect.Type, defaultDefName string) string {
				return t.Name()
			}),
		)
	}

	s.reflector = r
	s.specMarshaller = r.Spec
	s.addToComponents = func(name string, sch jsonschema.Schema) {
		sm, err := sch.ToSchemaOrBool().ToSimpleMap()
		if err != nil {
			return
		}
		r.SpecEns().ComponentsEns().WithSchemasItem(name, sm)
	}

	return nil
}

func wrapNullableRefInAnyOf(rc *jsonschema.ReflectContext) {
	rc.InterceptNullability = func(params jsonschema.InterceptNullabilityParams) {
		schema := params.Schema
		if schema.Ref != nil && schema.HasType(jsonschema.Null) {
			schema.RemoveType(jsonschema.Null)
			schema.Type = nil
			refSchema := *schema
			*schema = jsonschema.Schema{}
			schema.AnyOf = []jsonschema.SchemaOrBool{
				jsonschema.Null.ToSchemaOrBool(),
				refSchema.ToSchemaOrBool(),
			}
		}
	}
}

func (s *openapiService) AddOperation(method, pathPattern string, opts ...OperationOption) error {
	oc, err := s.reflector.NewOperationContext(method, pathPattern)
	if err != nil {
		return fmt.Errorf("create operation context: %w", err)
	}

	for _, opt := range opts {
		opt(oc)
	}

	if err := s.reflector.AddOperation(oc); err != nil {
		return fmt.Errorf("add operation: %w", err)
	}

	return nil
}

func (s *openapiService) SpecJSON() ([]byte, error) {
	return s.specMarshaller.MarshalJSON()
}

func (s *openapiService) SpecYAML() ([]byte, error) {
	return s.specMarshaller.MarshalYAML()
}

func (s *openapiService) RegisterSchema(name string, typ any) error {
	jsr := s.reflector.JSONSchemaReflector()

	sch, err := jsr.Reflect(typ,
		jsonschema.DefinitionsPrefix("#/components/schemas/"),
		jsonschema.CollectDefinitions(func(defName string, defSchema jsonschema.Schema) {
			s.addToComponents(defName, defSchema)
		}),
	)
	if err != nil {
		return fmt.Errorf("reflect schema: %w", err)
	}

	s.addToComponents(name, sch)

	return nil
}

var (
	_ OpenAPIService = (*openapiService)(nil)

	// compile-time checks that version-specific specs satisfy the marshaller.
	_ specMarshaller = (*openapi3.Spec)(nil)
	_ specMarshaller = (*openapi31.Spec)(nil)
)
