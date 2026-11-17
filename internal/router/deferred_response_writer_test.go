package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errWrite = errors.New("forced write error")

// failingResponseWriter lets tests exercise the wrapped-writer write error paths.
type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *failingResponseWriter) WriteHeader(int) {}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

func newDeferredResponseWriter() *DeferredResponseWriter {
	return &DeferredResponseWriter{
		Wrapped: httptest.NewRecorder(),
		Logger:  &internaltests.MockLogger{},
	}
}

func TestFlushPreservesCookies(t *testing.T) {
	// Create a mock ResponseWriter
	originalWriter := httptest.NewRecorder()

	// Create DeferredResponseWriter
	drw := &DeferredResponseWriter{
		Wrapped: originalWriter,
		Logger:  &internaltests.MockLogger{},
	}

	// Set a cookie directly on the original ResponseWriter (like CSRF plugin does)
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "csrf_token",
		Value: "test-token-123",
		Path:  "/",
	})

	// Create a RequestContext with JSON response (like CSRF validation does)
	reqCtx := &models.RequestContext{
		ResponseStatus:  http.StatusForbidden,
		ResponseHeaders: make(http.Header),
		ResponseBody:    []byte(`{"message": "invalid csrf token"}`),
		ResponseReady:   true,
	}
	reqCtx.ResponseHeaders.Set("Content-Type", "application/json")

	// Override with context (simulating SetJSONResponse flow)
	drw.OverrideWithContext(reqCtx)

	// Flush the response
	err := drw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Check that both the CSRF cookie and JSON response headers are present
	result := originalWriter.Result()

	// Verify CSRF cookie is preserved
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
	}
	if len(cookies) > 0 && cookies[0].Name != "csrf_token" {
		t.Errorf("Expected cookie name 'csrf_token', got '%s'", cookies[0].Name)
	}
	if len(cookies) > 0 && cookies[0].Value != "test-token-123" {
		t.Errorf("Expected cookie value 'test-token-123', got '%s'", cookies[0].Value)
	}

	// Verify JSON content type header is set
	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify status code
	if result.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, result.StatusCode)
	}
}

func TestDeferredResponseWriterStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		write      func(w *DeferredResponseWriter)
		wantStatus int
	}{
		{
			name: "reports status after WriteHeader",
			write: func(w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "reports implicit 200 after Write",
			write: func(w *DeferredResponseWriter) {
				_, _ = w.Write([]byte("ok"))
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "reports zero before any write",
			write:      func(w *DeferredResponseWriter) {},
			wantStatus: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := &DeferredResponseWriter{
				Wrapped: httptest.NewRecorder(),
				Logger:  &internaltests.MockLogger{},
			}

			tc.write(drw)

			if got := drw.StatusCode(); got != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, got)
			}
		})
	}
}

func TestFlushPreservesMultipleCookies(t *testing.T) {
	// Create a mock ResponseWriter
	originalWriter := httptest.NewRecorder()

	// Create DeferredResponseWriter
	drw := &DeferredResponseWriter{
		Wrapped: originalWriter,
		Logger:  &internaltests.MockLogger{},
	}

	// Set multiple cookies directly on the original ResponseWriter
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "csrf_token",
		Value: "test-token-123",
		Path:  "/",
	})
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "session_id",
		Value: "session-456",
		Path:  "/",
	})

	// Create a RequestContext with JSON response that also sets a cookie
	reqCtx := &models.RequestContext{
		ResponseStatus:  http.StatusOK,
		ResponseHeaders: make(http.Header),
		ResponseBody:    []byte(`{"success": true}`),
		ResponseReady:   true,
	}
	reqCtx.ResponseHeaders.Set("Content-Type", "application/json")
	reqCtx.ResponseHeaders["Set-Cookie"] = []string{"new_cookie=new_value; Path=/"}

	// Override with context
	drw.OverrideWithContext(reqCtx)

	// Flush the response
	err := drw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Check that all cookies are present
	result := originalWriter.Result()
	cookies := result.Cookies()

	// Should have 3 cookies total (2 original + 1 override)
	if len(cookies) != 3 {
		t.Errorf("Expected 3 cookies, got %d", len(cookies))
	}

	// Verify specific cookies exist
	cookieMap := make(map[string]string)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	if cookieMap["csrf_token"] != "test-token-123" {
		t.Errorf("CSRF token not preserved correctly")
	}
	if cookieMap["session_id"] != "session-456" {
		t.Errorf("Session ID not preserved correctly")
	}
	if cookieMap["new_cookie"] != "new_value" {
		t.Errorf("New cookie not set correctly")
	}
}

func TestDeferredResponseWriterRequestContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  *models.RequestContext
	}{
		{
			name: "stores nil context",
			ctx:  nil,
		},
		{
			name: "stores request context",
			ctx: &models.RequestContext{
				Path:   "/sign-in",
				Method: http.MethodPost,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := newDeferredResponseWriter()
			drw.SetRequestContext(tc.ctx)

			assert.Same(t, tc.ctx, drw.GetRequestContext())
		})
	}
}

func TestDeferredResponseWriterHeader(t *testing.T) {
	t.Parallel()

	drw := newDeferredResponseWriter()

	drw.Header().Set("X-Request-ID", "req-1")

	recorder, ok := drw.Wrapped.(*httptest.ResponseRecorder)
	require.True(t, ok)
	assert.Equal(t, "req-1", recorder.Header().Get("X-Request-ID"))
	assert.Equal(t, "req-1", recorder.Result().Header.Get("X-Request-ID"))
}

func TestDeferredResponseWriterWriteHeaderIsIdempotent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		statuses     []int
		wantStatus   int
		wantRecorded int
		writeBody    string
		wantBody     string
	}{
		{
			name:         "keeps first status when called twice",
			statuses:     []int{http.StatusCreated, http.StatusInternalServerError},
			wantStatus:   http.StatusCreated,
			wantRecorded: http.StatusCreated,
		},
		{
			name:         "keeps explicit status when body follows",
			statuses:     []int{http.StatusAccepted},
			wantStatus:   http.StatusAccepted,
			wantRecorded: http.StatusAccepted,
			writeBody:    "accepted",
			wantBody:     "accepted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := newDeferredResponseWriter()
			for _, status := range tc.statuses {
				drw.WriteHeader(status)
			}
			if tc.writeBody != "" {
				_, err := drw.Write([]byte(tc.writeBody))
				require.NoError(t, err)
			}
			require.NoError(t, drw.Flush())

			assert.Equal(t, tc.wantStatus, drw.StatusCode())

			recorder, ok := drw.Wrapped.(*httptest.ResponseRecorder)
			require.True(t, ok)
			assert.Equal(t, tc.wantRecorded, recorder.Code)
			assert.Equal(t, tc.wantBody, recorder.Body.String())
		})
	}
}

func TestDeferredResponseWriterFlush(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		write      func(t *testing.T, w *DeferredResponseWriter)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty response writes nothing",
			write:      func(t *testing.T, w *DeferredResponseWriter) {},
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		{
			name: "buffered body uses implicit 200",
			write: func(t *testing.T, w *DeferredResponseWriter) {
				_, err := w.Write([]byte("hello"))
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "hello",
		},
		{
			name: "explicit status with body",
			write: func(t *testing.T, w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, err := w.Write([]byte(`{"id":"1"}`))
				require.NoError(t, err)
			},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":"1"}`,
		},
		{
			name: "explicit status without body",
			write: func(t *testing.T, w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantStatus: http.StatusNoContent,
			wantBody:   "",
		},
		{
			name: "write after WriteHeader keeps explicit status",
			write: func(t *testing.T, w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusAccepted)
				_, err := w.Write([]byte("queued"))
				require.NoError(t, err)
				assert.Equal(t, http.StatusAccepted, w.StatusCode())
			},
			wantStatus: http.StatusAccepted,
			wantBody:   "queued",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := newDeferredResponseWriter()
			tc.write(t, drw)

			require.NoError(t, drw.Flush())

			recorder, ok := drw.Wrapped.(*httptest.ResponseRecorder)
			require.True(t, ok)
			assert.Equal(t, tc.wantStatus, recorder.Code)
			assert.Equal(t, tc.wantBody, recorder.Body.String())
		})
	}
}

func TestDeferredResponseWriterFlushOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prepare    func(t *testing.T, w *DeferredResponseWriter)
		ctx        *models.RequestContext
		wantStatus int
		wantBody   string
		wantHeader map[string]string
	}{
		{
			name:    "applies override headers body and status",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {},
			ctx: &models.RequestContext{
				ResponseStatus:  http.StatusUnauthorized,
				ResponseHeaders: http.Header{"Content-Type": []string{"application/json"}},
				ResponseBody:    []byte(`{"message":"unauthorized"}`),
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"message":"unauthorized"}`,
			wantHeader: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:    "nil override headers and body keep empty response",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {},
			ctx: &models.RequestContext{
				ResponseStatus: http.StatusTeapot,
			},
			wantStatus: http.StatusTeapot,
			wantBody:   "",
		},
		{
			name: "zero override status falls back to captured status",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusAccepted)
				_, err := w.Write([]byte("buffered"))
				require.NoError(t, err)
			},
			ctx: &models.RequestContext{
				ResponseHeaders: http.Header{"X-Override": []string{"1"}},
			},
			wantStatus: http.StatusAccepted,
			wantBody:   "",
			wantHeader: map[string]string{"X-Override": "1"},
		},
		{
			name:    "zero override status without captured status uses 200",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {},
			ctx: &models.RequestContext{
				ResponseBody: []byte("override"),
			},
			wantStatus: http.StatusOK,
			wantBody:   "override",
		},
		{
			name: "override status replaces captured status",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {
				w.WriteHeader(http.StatusAccepted)
			},
			ctx: &models.RequestContext{
				ResponseStatus: http.StatusConflict,
			},
			wantStatus: http.StatusConflict,
			wantBody:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := newDeferredResponseWriter()
			tc.prepare(t, drw)
			drw.OverrideWithContext(tc.ctx)

			require.NoError(t, drw.Flush())

			recorder, ok := drw.Wrapped.(*httptest.ResponseRecorder)
			require.True(t, ok)
			assert.Equal(t, tc.wantStatus, recorder.Code)
			assert.Equal(t, tc.wantBody, recorder.Body.String())
			for key, value := range tc.wantHeader {
				assert.Equal(t, value, recorder.Header().Get(key))
			}
		})
	}
}

func TestDeferredResponseWriterFlushMergesCookies(t *testing.T) {
	t.Parallel()

	drw := newDeferredResponseWriter()

	http.SetCookie(drw.Wrapped, &http.Cookie{Name: "csrf_token", Value: "original", Path: "/"})
	http.SetCookie(drw.Wrapped, &http.Cookie{Name: "session_id", Value: "session-1", Path: "/"})

	drw.OverrideWithContext(&models.RequestContext{
		ResponseStatus: http.StatusOK,
		ResponseHeaders: http.Header{
			"Content-Type": []string{"application/json"},
			"Set-Cookie": []string{
				"csrf_token=overridden; Path=/",
				"new_cookie=new_value; Path=/",
			},
		},
		ResponseBody: []byte(`{"success":true}`),
	})

	require.NoError(t, drw.Flush())

	recorder, ok := drw.Wrapped.(*httptest.ResponseRecorder)
	require.True(t, ok)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 3)

	values := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		values[cookie.Name] = cookie.Value
	}
	assert.Equal(t, "original", values["csrf_token"], "duplicate cookie name must keep the existing cookie")
	assert.Equal(t, "session-1", values["session_id"])
	assert.Equal(t, "new_value", values["new_cookie"])
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestDeferredResponseWriterFlushWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(t *testing.T, w *DeferredResponseWriter)
	}{
		{
			name: "override body write error",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {
				w.OverrideWithContext(&models.RequestContext{
					ResponseStatus: http.StatusOK,
					ResponseBody:   []byte("override"),
				})
			},
		},
		{
			name: "buffered body write error",
			prepare: func(t *testing.T, w *DeferredResponseWriter) {
				_, err := w.Write([]byte("buffered"))
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			drw := &DeferredResponseWriter{
				Wrapped: &failingResponseWriter{},
				Logger:  &internaltests.MockLogger{},
			}
			tc.prepare(t, drw)

			require.ErrorIs(t, drw.Flush(), errWrite)
		})
	}

	t.Run("override without body ignores wrapped write error", func(t *testing.T) {
		t.Parallel()

		drw := &DeferredResponseWriter{
			Wrapped: &failingResponseWriter{},
			Logger:  &internaltests.MockLogger{},
		}
		drw.OverrideWithContext(&models.RequestContext{ResponseStatus: http.StatusOK})

		require.NoError(t, drw.Flush())
	})
}

func TestExtractCookieName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "extracts name before first equals",
			header: "csrf_token=abc; Path=/",
			want:   "csrf_token",
		},
		{
			name:   "keeps only the leading name when value has equals",
			header: "token=a=b",
			want:   "token",
		},
		{
			name:   "returns input when separator is missing",
			header: "csrf_token",
			want:   "csrf_token",
		},
		{
			name:   "returns empty input unchanged",
			header: "",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, extractCookieName(tc.header))
		})
	}
}
