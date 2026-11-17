package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestGenerateUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(t *testing.T, first, second string)
	}{
		{
			name: "generates unique values",
			check: func(t *testing.T, first, second string) {
				assert.NotEqual(t, first, second)
			},
		},
		{
			name: "generates version 4 UUIDs",
			check: func(t *testing.T, first, _ string) {
				parsed, err := uuid.Parse(first)
				require.NoError(t, err)
				assert.Equal(t, uuid.Version(4), parsed.Version())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.check(t, GenerateUUID(), GenerateUUID())
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercases and trims", input: "  User@Example.COM \t", want: "user@example.com"},
		{name: "already normalized", input: "user@example.com", want: "user@example.com"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, NormalizeEmail(tt.input))
		})
	}
}

func TestDerefPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{name: "nil returns zero value", input: nil, want: ""},
		{name: "returns pointed value", input: new("value"), want: "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, DerefPointer(tt.input))
		})
	}
}

func TestCompareStringArrays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{name: "equal order", a: []string{"a", "b"}, b: []string{"a", "b"}, want: true},
		{name: "different length", a: []string{"a"}, b: []string{"a", "b"}, want: false},
		{name: "different order", a: []string{"a", "b"}, b: []string{"b", "a"}, want: false},
		{name: "nil and empty are equal", a: nil, b: []string{}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, CompareStringArrays(tt.a, tt.b))
		})
	}
}

func TestCloneRequestContext(t *testing.T) {
	t.Parallel()

	newFullRequestContext := func() *models.RequestContext {
		return &models.RequestContext{
			Request:         httptest.NewRequest(http.MethodGet, "/me", nil),
			ResponseWriter:  httptest.NewRecorder(),
			Path:            "/me",
			Method:          http.MethodGet,
			Headers:         http.Header{"X-Test": []string{"value"}},
			Route:           &models.Route{Method: http.MethodGet, Path: "/me"},
			Actor:           &models.Actor{ID: "user-1"},
			Handled:         true,
			ClientIP:        "1.2.3.4",
			Values:          map[string]any{"key": "value"},
			ResponseHeaders: http.Header{"X-Response": []string{"a", "b"}},
			ResponseStatus:  http.StatusCreated,
			ResponseBody:    []byte("body"),
			ResponseReady:   true,
		}
	}

	tests := []struct {
		name     string
		original *models.RequestContext
		verify   func(t *testing.T, original, cloned *models.RequestContext)
	}{
		{
			name: "nil context returns nil",
			verify: func(t *testing.T, _, cloned *models.RequestContext) {
				assert.Nil(t, cloned)
			},
		},
		{
			name:     "shares request, route and actor references",
			original: newFullRequestContext(),
			verify: func(t *testing.T, original, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				assert.Same(t, original.Request, cloned.Request)
				assert.Same(t, original.Route, cloned.Route)
				assert.Same(t, original.Actor, cloned.Actor)
			},
		},
		{
			name:     "copies path, method, headers and handled flag",
			original: newFullRequestContext(),
			verify: func(t *testing.T, original, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				assert.Equal(t, original.Path, cloned.Path)
				assert.Equal(t, original.Method, cloned.Method)
				assert.Equal(t, original.Headers, cloned.Headers)
				assert.True(t, cloned.Handled)
			},
		},
		{
			name:     "drops response state and client IP",
			original: newFullRequestContext(),
			verify: func(t *testing.T, _, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				assert.Nil(t, cloned.ResponseWriter)
				assert.Empty(t, cloned.ClientIP)
				assert.Zero(t, cloned.ResponseStatus)
				assert.Nil(t, cloned.ResponseBody)
				assert.False(t, cloned.ResponseReady)
			},
		},
		{
			name:     "detaches values map",
			original: newFullRequestContext(),
			verify: func(t *testing.T, original, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				cloned.Values["key"] = "changed"
				cloned.Values["extra"] = true
				assert.Equal(t, "value", original.Values["key"])
				assert.NotContains(t, original.Values, "extra")
			},
		},
		{
			name:     "detaches response headers",
			original: newFullRequestContext(),
			verify: func(t *testing.T, original, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				cloned.ResponseHeaders["X-Response"][0] = "changed"
				cloned.ResponseHeaders["X-New"] = []string{"new"}
				assert.Equal(t, []string{"a", "b"}, original.ResponseHeaders["X-Response"])
				assert.NotContains(t, original.ResponseHeaders, "X-New")
			},
		},
		{
			name:     "nil maps become empty maps",
			original: &models.RequestContext{},
			verify: func(t *testing.T, _, cloned *models.RequestContext) {
				require.NotNil(t, cloned)
				assert.NotNil(t, cloned.Values)
				assert.Empty(t, cloned.Values)
				assert.NotNil(t, cloned.ResponseHeaders)
				assert.Empty(t, cloned.ResponseHeaders)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.verify(t, tt.original, CloneRequestContext(tt.original))
		})
	}
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "adds leading slash", input: "users", want: "/users"},
		{name: "removes trailing slash", input: "/users/", want: "/users"},
		{name: "keeps nested path", input: "/users/me", want: "/users/me"},
		{name: "keeps duplicate inner slashes", input: "/users//me", want: "/users//me"},
		{name: "empty becomes empty", input: "", want: ""},
		{name: "root becomes empty", input: "/", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, NormalizePath(tt.input))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{name: "zero minutes", input: 0, want: "0 minutes"},
		{name: "one minute", input: time.Minute, want: "1 minute"},
		{name: "many minutes", input: 15 * time.Minute, want: "15 minutes"},
		{name: "minutes truncate seconds", input: 59*time.Minute + 59*time.Second, want: "59 minutes"},
		{name: "one hour", input: time.Hour, want: "1 hour"},
		{name: "many hours", input: 5 * time.Hour, want: "5 hours"},
		{name: "hours truncate minutes", input: 90 * time.Minute, want: "1 hour"},
		{name: "one day", input: 24 * time.Hour, want: "1 day"},
		{name: "many days", input: 48 * time.Hour, want: "2 days"},
		{name: "days truncate hours", input: 25 * time.Hour, want: "1 day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, FormatDuration(tt.input))
		})
	}
}

func TestReadStringSliceMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		reqCtx  *models.RequestContext
		key     string
		want    []string
		wantNil bool
	}{
		{name: "nil context", reqCtx: nil, key: "plugins", wantNil: true},
		{name: "nil route", reqCtx: &models.RequestContext{}, key: "plugins", wantNil: true},
		{name: "nil metadata", reqCtx: &models.RequestContext{Route: &models.Route{}}, key: "plugins", wantNil: true},
		{
			name:    "missing key",
			reqCtx:  &models.RequestContext{Route: &models.Route{Metadata: map[string]any{"other": []string{"x"}}}},
			key:     "plugins",
			wantNil: true,
		},
		{
			name:    "nil value",
			reqCtx:  &models.RequestContext{Route: &models.Route{Metadata: map[string]any{"plugins": nil}}},
			key:     "plugins",
			wantNil: true,
		},
		{
			name:   "string slice trimmed and filtered",
			reqCtx: &models.RequestContext{Route: &models.Route{Metadata: map[string]any{"plugins": []string{" jwt ", "", "  "}}}},
			key:    "plugins",
			want:   []string{"jwt"},
		},
		{
			name:   "any slice filters non strings",
			reqCtx: &models.RequestContext{Route: &models.Route{Metadata: map[string]any{"plugins": []any{" jwt ", 1, " totp ", ""}}}},
			key:    "plugins",
			want:   []string{"jwt", "totp"},
		},
		{
			name:    "unsupported type",
			reqCtx:  &models.RequestContext{Route: &models.Route{Metadata: map[string]any{"plugins": 42}}},
			key:     "plugins",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ReadStringSliceMetadata(tt.reqCtx, tt.key)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadStringSliceFromMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
		key      string
		want     []string
		wantNil  bool
	}{
		{name: "nil metadata", metadata: nil, key: "plugins", wantNil: true},
		{name: "missing key", metadata: map[string]any{"other": []string{"x"}}, key: "plugins", wantNil: true},
		{name: "nil value", metadata: map[string]any{"plugins": nil}, key: "plugins", wantNil: true},
		{
			name:     "string slice trimmed and filtered",
			metadata: map[string]any{"plugins": []string{" jwt ", "", "  "}},
			key:      "plugins",
			want:     []string{"jwt"},
		},
		{
			name:     "any slice filters non strings",
			metadata: map[string]any{"plugins": []any{"jwt", 1, " totp ", ""}},
			key:      "plugins",
			want:     []string{"jwt", "totp"},
		},
		{
			name:     "unsupported type",
			metadata: map[string]any{"plugins": 42},
			key:      "plugins",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ReadStringSliceFromMetadata(tt.metadata, tt.key)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeStringSlices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values [][]string
		want   []string
	}{
		{name: "no values", values: nil, want: []string{}},
		{
			name:   "trims and deduplicates preserving order",
			values: [][]string{{" a ", "b"}, {"b", " c ", ""}, {"a", "d"}},
			want:   []string{"a", "b", "c", "d"},
		},
		{name: "empty and whitespace only values", values: [][]string{{""}, {"   "}}, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, MergeStringSlices(tt.values...))
		})
	}
}

func TestNormalizeRoutePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "adds leading slash", input: "users/me", want: "/users/me"},
		{name: "removes trailing slash", input: "/users/me/", want: "/users/me"},
		{name: "collapses duplicate slashes", input: "users//me", want: "/users/me"},
		{name: "collapses leading duplicate slashes", input: "//users", want: "/users"},
		{name: "empty stays empty", input: "", want: ""},
		{name: "root becomes empty", input: "/", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, NormalizeRoutePattern(tt.input))
		})
	}
}

func TestDedent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "uniform indent", input: "  a\n  b", want: "a\nb\n"},
		{name: "minimum indent wins", input: "    a\n      b", want: "a\n  b\n"},
		{name: "no indent", input: "a\nb", want: "a\nb\n"},
		{name: "tabs", input: "\ta\n\tb", want: "a\nb\n"},
		{name: "blank lines stay empty", input: "  a\n\n  b", want: "a\n\nb\n"},
		{name: "whitespace only lines are emptied", input: "  a\n   \n  b", want: "a\n\nb\n"},
		{name: "surrounding newlines are trimmed", input: "\n  a\n", want: "a\n"},
		{name: "empty input becomes newline", input: "", want: "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Dedent(tt.input))
		})
	}
}
