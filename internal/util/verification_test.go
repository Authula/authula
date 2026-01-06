package util

import (
	"testing"
)

func TestBuildVerificationURL_WithCallback(t *testing.T) {
	baseURL := "https://example.com"
	basePath := "/auth"
	token := "abc123"
	callback := "https://app.com/callback?token=abc123"
	expected := "https://example.com/auth/verify-email?callback_url=https%3A%2F%2Fapp.com%2Fcallback%3Ftoken%3Dabc123&token=abc123"

	result := BuildVerificationURL(baseURL, basePath, token, &callback)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildVerificationURL_WithoutCallback(t *testing.T) {
	baseURL := "https://example.com"
	basePath := "/auth"
	token := "xyz789"
	expected := "https://example.com/auth/verify-email?token=xyz789"

	result := BuildVerificationURL(baseURL, basePath, token, nil)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildVerificationURL_EmptyCallback(t *testing.T) {
	baseURL := "https://example.com"
	basePath := "/auth"
	token := "token"
	emptyCallback := ""
	expected := "https://example.com/auth/verify-email?token=token"

	result := BuildVerificationURL(baseURL, basePath, token, &emptyCallback)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildVerificationURL_BasePathTrailingSlash(t *testing.T) {
	baseURL := "https://example.com"
	basePath := "/auth/"
	token := "token"
	expected := "https://example.com/auth/verify-email?token=token"

	result := BuildVerificationURL(baseURL, basePath, token, nil)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
