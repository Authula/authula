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

	t.Run("decodes body", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"user","age":30}`))

		var got jsonTestPayload
		err := ParseJSON(request, &got)

		require.NoError(t, err)
		assert.Equal(t, jsonTestPayload{Name: "user", Age: 30}, got)
	})

	t.Run("malformed body returns error", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))

		var got jsonTestPayload
		err := ParseJSON(request, &got)

		require.Error(t, err)
	})
}

func TestJSONResponse(t *testing.T) {
	t.Parallel()

	t.Run("writes status header and body", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		JSONResponse(recorder, http.StatusCreated, jsonTestPayload{Name: "user", Age: 30})

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var got jsonTestPayload
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, jsonTestPayload{Name: "user", Age: 30}, got)
	})

	t.Run("encoding failure writes error details", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		JSONResponse(recorder, http.StatusOK, make(chan int))

		assert.Contains(t, recorder.Body.String(), "unsupported type")
	})
}

func TestToMap(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, ToMap(nil))
	})

	t.Run("map is returned unchanged", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{"key": "value"}

		got := ToMap(input)

		assert.Equal(t, input, got)
		got["key"] = "changed"
		assert.Equal(t, "changed", input["key"])
	})

	t.Run("struct uses json tags", func(t *testing.T) {
		t.Parallel()

		got := ToMap(jsonTestPayload{Name: "user", Age: 30})

		assert.Equal(t, map[string]any{"name": "user", "age": float64(30)}, got)
	})

	t.Run("unmarshallable value returns nil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, ToMap(make(chan int)))
	})

	t.Run("non object json returns nil map", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, ToMap("just-a-string"))
	})
}

func TestFromMap(t *testing.T) {
	t.Parallel()

	t.Run("nil map returns nil", func(t *testing.T) {
		t.Parallel()

		var got jsonTestPayload
		require.NoError(t, FromMap(nil, &got))
		assert.Equal(t, jsonTestPayload{}, got)
	})

	t.Run("decodes into struct", func(t *testing.T) {
		t.Parallel()

		var got jsonTestPayload
		err := FromMap(map[string]any{"name": "user", "age": 30}, &got)

		require.NoError(t, err)
		assert.Equal(t, jsonTestPayload{Name: "user", Age: 30}, got)
	})

	t.Run("marshal error is returned", func(t *testing.T) {
		t.Parallel()

		var got jsonTestPayload
		err := FromMap(map[string]any{"bad": make(chan int)}, &got)

		require.Error(t, err)
	})

	t.Run("unmarshal error is returned", func(t *testing.T) {
		t.Parallel()

		var got jsonTestPayload
		err := FromMap(map[string]any{"age": "not-a-number"}, &got)

		require.Error(t, err)
	})
}
