package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonTestPayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestParseJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		want    jsonTestPayload
		wantErr bool
	}{
		{
			name: "decodes body",
			body: `{"name":"user","age":30}`,
			want: jsonTestPayload{Name: "user", Age: 30},
		},
		{
			name:    "malformed body returns error",
			body:    `{"name":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))

			var got jsonTestPayload
			err := ParseJSON(request, &got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJSONResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		data   any
		check  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "writes status header and body",
			status: http.StatusCreated,
			data:   jsonTestPayload{Name: "user", Age: 30},
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var got jsonTestPayload
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
				assert.Equal(t, jsonTestPayload{Name: "user", Age: 30}, got)
			},
		},
		{
			name:   "encoding failure writes error details",
			status: http.StatusOK,
			data:   make(chan int),
			check: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Contains(t, recorder.Body.String(), "unsupported type")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			JSONResponse(recorder, tt.status, tt.data)

			tt.check(t, recorder)
		})
	}
}

func TestToMap(t *testing.T) {
	t.Parallel()

	input := map[string]any{"key": "value"}

	tests := []struct {
		name  string
		input any
		want  map[string]any
		check func(t *testing.T, got map[string]any)
	}{
		{
			name:  "nil returns nil",
			input: nil,
		},
		{
			name:  "map is returned unchanged",
			input: input,
			want:  input,
			check: func(t *testing.T, got map[string]any) {
				got["key"] = "changed"
				assert.Equal(t, "changed", input["key"])
			},
		},
		{
			name:  "struct uses json tags",
			input: jsonTestPayload{Name: "user", Age: 30},
			want:  map[string]any{"name": "user", "age": float64(30)},
		},
		{
			name:  "unmarshallable value returns nil",
			input: make(chan int),
		},
		{
			name:  "non object json returns nil map",
			input: "just-a-string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToMap(tt.input)

			assert.Equal(t, tt.want, got)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   map[string]any
		want    jsonTestPayload
		wantErr bool
	}{
		{
			name:  "nil map returns nil",
			input: nil,
		},
		{
			name:  "decodes into struct",
			input: map[string]any{"name": "user", "age": 30},
			want:  jsonTestPayload{Name: "user", Age: 30},
		},
		{
			name:    "marshal error is returned",
			input:   map[string]any{"bad": make(chan int)},
			wantErr: true,
		},
		{
			name:    "unmarshal error is returned",
			input:   map[string]any{"age": "not-a-number"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got jsonTestPayload
			err := FromMap(tt.input, &got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
