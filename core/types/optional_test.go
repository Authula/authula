package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
