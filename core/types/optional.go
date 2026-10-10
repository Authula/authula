package types

import (
	"encoding/json"
	"reflect"

	"github.com/swaggest/jsonschema-go"
)

// Optional represents a JSON object field that may be absent, present with a
// null value, or present with a concrete value. The zero value is "absent".
//
// It is used on PATCH-style request structs to tell an omitted field apart from
// a field explicitly set to null, which a plain pointer cannot do.
//
// Use the `omitzero` JSON tag so an absent field is left out when encoding.
type Optional[T any] struct {
	Value   T
	Present bool
}

// UnmarshalJSON records that the field was present and decodes its value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Present = true

	var zero T
	o.Value = zero

	return json.Unmarshal(data, &o.Value)
}

// MarshalJSON encodes the underlying value. An absent optional is omitted from
// the object when the field uses the `omitzero` tag.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// IsZero reports whether the field was absent so that encoding/json can omit it
// when the `omitzero` option is used.
func (o Optional[T]) IsZero() bool {
	return !o.Present
}

// PrepareJSONSchema makes the generated schema describe the inner type T
// instead of the wrapper struct. This keeps Optional invisible to API clients.
//
// It works for scalar, pointer, and map inner types. Named struct inner types
// can produce references that the outer reflector does not collect.
func (o Optional[T]) PrepareJSONSchema(schema *jsonschema.Schema) error {
	innerType := reflect.TypeFor[T]()

	var reflector jsonschema.Reflector
	reflector.InlineDefinition(reflect.Zero(innerType).Interface())

	inner, err := reflector.Reflect(reflect.Zero(innerType).Interface())
	if err != nil {
		return err
	}

	*schema = inner

	return nil
}
