package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"

	coretypes "github.com/Authula/authula/core/types"
)

func TestOptional_Unmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantPresent bool
		wantValue   *string
	}{
		{name: "absent", body: `{}`, wantPresent: false, wantValue: nil},
		{name: "null", body: `{"logo":null}`, wantPresent: true, wantValue: nil},
		{name: "empty string", body: `{"logo":""}`, wantPresent: true, wantValue: new("")},
		{name: "value", body: `{"logo":"http://some/url/logo.svg"}`, wantPresent: true, wantValue: new("http://some/url/logo.svg")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var payload struct {
				Logo coretypes.Optional[*string] `json:"logo,omitzero"`
			}
			require.NoError(t, json.Unmarshal([]byte(tt.body), &payload))

			assert.Equal(t, tt.wantPresent, payload.Logo.Present)
			assert.Equal(t, tt.wantValue, payload.Logo.Value)
		})
	}
}

func TestOptional_Marshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value coretypes.Optional[*string]
		want  string
	}{
		{name: "absent is omitted", value: coretypes.Optional[*string]{}, want: `{}`},
		{name: "null is kept", value: coretypes.Optional[*string]{Present: true, Value: nil}, want: `{"logo":null}`},
		{name: "value is kept", value: coretypes.Optional[*string]{Present: true, Value: new("http://some/url/logo.svg")}, want: `{"logo":"http://some/url/logo.svg"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := struct {
				Logo coretypes.Optional[*string] `json:"logo,omitzero"`
			}{Logo: tt.value}

			data, err := json.Marshal(payload)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestOptional_PrepareJSONSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		prepare      func(*jsonschema.Schema) error
		wantTypes    []jsonschema.SimpleType
		wantValueTyp *jsonschema.SimpleType
	}{
		{
			name: "string",
			prepare: func(schema *jsonschema.Schema) error {
				return coretypes.Optional[string]{}.PrepareJSONSchema(schema)
			},
			wantTypes: []jsonschema.SimpleType{jsonschema.String},
		},
		{
			name: "pointer to string",
			prepare: func(schema *jsonschema.Schema) error {
				return coretypes.Optional[*string]{}.PrepareJSONSchema(schema)
			},
			wantTypes: []jsonschema.SimpleType{jsonschema.Null, jsonschema.String},
		},
		{
			name: "map of strings",
			prepare: func(schema *jsonschema.Schema) error {
				return coretypes.Optional[map[string]string]{}.PrepareJSONSchema(schema)
			},
			wantTypes:    []jsonschema.SimpleType{jsonschema.Object},
			wantValueTyp: new(jsonschema.String),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema jsonschema.Schema
			require.NoError(t, tt.prepare(&schema))

			require.NotNil(t, schema.Type)
			assert.Equal(t, tt.wantTypes, schemaTypeValues(schema.Type))
			assert.Empty(t, schema.Properties, "Optional wrapper fields must not leak into the schema")

			if tt.wantValueTyp != nil {
				require.NotNil(t, schema.AdditionalProperties)
				require.NotNil(t, schema.AdditionalProperties.TypeObject)
				require.NotNil(t, schema.AdditionalProperties.TypeObject.Type)
				assert.Equal(t,
					[]jsonschema.SimpleType{*tt.wantValueTyp},
					schemaTypeValues(schema.AdditionalProperties.TypeObject.Type),
				)
			}
		})
	}
}

func TestOptional_PrepareJSONSchema_UnsupportedInnerType(t *testing.T) {
	t.Parallel()

	var schema jsonschema.Schema
	err := coretypes.Optional[chan int]{}.PrepareJSONSchema(&schema)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "type is not supported")
	assert.Nil(t, schema.Type)
}

func schemaTypeValues(typ *jsonschema.Type) []jsonschema.SimpleType {
	if typ == nil {
		return nil
	}
	if typ.SimpleTypes != nil {
		return []jsonschema.SimpleType{*typ.SimpleTypes}
	}

	return typ.SliceOfSimpleTypeValues
}
