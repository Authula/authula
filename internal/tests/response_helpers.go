package tests

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func DecodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode json response: %v body=%s", err, w.Body.String())
	}
	return envelope
}

func GetMap(t *testing.T, envelope map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := envelope[key].(map[string]any)
	if !ok {
		t.Fatalf("expected %q object in response, got %v", key, envelope[key])
	}
	return value
}

func GetArray(t *testing.T, envelope map[string]any, key string) []any {
	t.Helper()
	value, ok := envelope[key].([]any)
	if !ok {
		t.Fatalf("expected %q array in response, got %v", key, envelope[key])
	}
	return value
}

func GetString(t *testing.T, envelope map[string]any, key string) string {
	t.Helper()
	value, ok := envelope[key].(string)
	if !ok {
		t.Fatalf("expected %q string in response, got %v", key, envelope[key])
	}
	return value
}

func AssertHasKey(t *testing.T, envelope map[string]any, key string) {
	t.Helper()
	if _, ok := envelope[key]; !ok {
		t.Fatalf("expected response to include key %q, got %v", key, envelope)
	}
}
