package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

type nullableInner struct {
	Value string `json:"value"`
}

type nullableOuter struct {
	Inner nullableInner `json:"inner" nullable:"true"`
	Plain nullableInner `json:"plain"`
	Name  *string       `json:"name"`
}

type nestedParent struct {
	Inner nullableInner `json:"inner"`
}

type floatWithNaNDefault struct {
	Ratio float64 `json:"ratio" default:"NaN"`
}

func decodeSpecSchemas(t *testing.T, svc openapi.OpenAPIService) map[string]any {
	t.Helper()

	spec, err := svc.SpecJSON()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(spec, &result))

	components, _ := result["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)

	return schemas
}

func TestRegisterSchema_NullableProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		property string
		want     any
	}{
		{
			name:     "nullable named struct is wrapped in anyOf with null",
			property: "inner",
			want: map[string]any{
				"anyOf": []any{
					map[string]any{"type": "null"},
					map[string]any{"$ref": "#/components/schemas/nullableInner"},
				},
			},
		},
		{
			name:     "non-nullable named struct keeps its plain reference",
			property: "plain",
			want:     map[string]any{"$ref": "#/components/schemas/nullableInner"},
		},
		{
			name:     "nullable primitive keeps its null type without a reference",
			property: "name",
			want:     map[string]any{"type": []any{"null", "string"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "",
				openapi.WithOpenAPIVersion("3.1.0"),
				openapi.WithShortSchemaNames())
			require.NoError(t, err)

			// Act
			err = svc.RegisterSchema("NullableOuter", &nullableOuter{})

			// Assert
			require.NoError(t, err)
			schemas := decodeSpecSchemas(t, svc)
			properties := schemas["NullableOuter"].(map[string]any)["properties"].(map[string]any)
			assert.Equal(t, tt.want, properties[tt.property])
		})
	}
}

func TestRegisterSchema_ReturnsErrorForUnsupportedType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  any
	}{
		{
			name: "channel type",
			typ:  make(chan int),
		},
		{
			name: "function type",
			typ:  func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "")
			require.NoError(t, err)

			// Act
			err = svc.RegisterSchema("Unsupported", tt.typ)

			// Assert
			assert.ErrorContains(t, err, "reflect schema:")
			assert.ErrorContains(t, err, "type is not supported")
		})
	}
}

func TestRegisterSchema_CollectsNestedDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "OpenAPI 3.0.3",
			version: "3.0.3",
		},
		{
			name:    "OpenAPI 3.1.0",
			version: "3.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "",
				openapi.WithOpenAPIVersion(tt.version),
				openapi.WithShortSchemaNames())
			require.NoError(t, err)

			// Act
			err = svc.RegisterSchema("NestedParent", &nestedParent{})

			// Assert
			require.NoError(t, err)
			schemas := decodeSpecSchemas(t, svc)
			assert.Contains(t, schemas, "NestedParent")
			assert.Contains(t, schemas, "nullableInner", "nested named type should be collected as a component")
		})
	}
}

// Pins current behavior: a schema whose default cannot be encoded is dropped from the 3.1
// components without an error. Update this test if RegisterSchema starts returning that error.
func TestRegisterSchema_OmitsSchemaWhoseDefaultCannotBeEncodedIn31(t *testing.T) {
	t.Parallel()

	// Arrange
	svc, err := openapi.NewOpenAPIService("Test", "1.0.0", "", openapi.WithOpenAPIVersion("3.1.0"))
	require.NoError(t, err)

	// Act
	err = svc.RegisterSchema("RatioSchema", &floatWithNaNDefault{})

	// Assert
	require.NoError(t, err)
	assert.NotContains(t, decodeSpecSchemas(t, svc), "RatioSchema")
}
