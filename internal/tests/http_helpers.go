package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func JSONRequest(t *testing.T, handler http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal json payload: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func Request(t *testing.T, handler http.Handler, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func ConcreteRoutePath(path string) string {
	if !strings.Contains(path, "{") {
		return path
	}

	var builder strings.Builder
	insideParam := false
	for _, r := range path {
		switch r {
		case '{':
			insideParam = true
			builder.WriteString("test-id")
		case '}':
			insideParam = false
		default:
			if !insideParam {
				builder.WriteRune(r)
			}
		}
	}

	return builder.String()
}
