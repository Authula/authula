package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/config"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/stretchr/testify/assert"
)

func TestEndpointHooksMiddleware_BeforeHook_ModifyRequestHeader(t *testing.T) {
	// Setup
	config := config.NewConfig(
		config.WithEndpointHooks(
			models.EndpointHooksConfig{
				Before: func(ctx *models.EndpointHookContext) error {
					// Set a request header that the handler will read
					ctx.Request.Header.Set("X-Test-From-Before", "from-before")
					return nil
				},
			},
		),
	)

	middleware := EndpointHooksMiddleware(config, nil) // authService is nil, ensure no session cookie

	handler := middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handler should be able to read the header set by the Before hook
			v := r.Header.Get("X-Test-From-Before")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(v))
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "from-before", rec.Body.String())
}

func TestEndpointHooksMiddleware_Response_JSONToHTML(t *testing.T) {
	// Setup
	config := config.NewConfig(
		config.WithEndpointHooks(models.EndpointHooksConfig{
			Response: func(ctx *models.EndpointHookContext) error {
				// Check if original response was JSON
				if len(ctx.ResponseHeaders["Content-Type"]) > 0 && ctx.ResponseHeaders["Content-Type"][0] == "application/json" {
					var response struct {
						Message string `json:"message"`
					}
					if err := json.Unmarshal(ctx.ResponseBody, &response); err != nil {
						return err
					}

					// Replace with HTML response
					ctx.ResponseBody = fmt.Appendf(nil, "<html><body><h1>%s</h1></body></html>", response.Message)
					ctx.ResponseHeaders["Content-Type"] = []string{"text/html"}
				}
				return nil
			},
		}),
	)

	middleware := EndpointHooksMiddleware(config, nil)

	handler := middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "OK"}`))
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html", rec.Header().Get("Content-Type"))
	assert.Equal(t, "<html><body><h1>OK</h1></body></html>", rec.Body.String())
}

func TestEndpointHooksMiddleware_Response_Error(t *testing.T) {
	// Setup
	config := config.NewConfig(
		config.WithEndpointHooks(models.EndpointHooksConfig{
			Response: func(ctx *models.EndpointHookContext) error {
				return assert.AnError
			},
		}),
	)

	middleware := EndpointHooksMiddleware(config, nil)

	handler := middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"message": "Internal Server Error"}`, rec.Body.String())
}
