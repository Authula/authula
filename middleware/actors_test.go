package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

// newActorRequest builds a request whose context already carries a
// RequestContext, as the router does before running middleware.
func newActorRequest(actor *models.Actor) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
		Method:         req.Method,
		Path:           req.URL.Path,
		Actor:          actor,
	}

	return req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))
}

func TestRequireAuthenticated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		actor        *models.Actor
		wantNext     bool
		wantStatus   int
		wantHandled  bool
		wantResponse string
	}{
		{
			name:         "rejects a request without an actor",
			actor:        nil,
			wantNext:     false,
			wantStatus:   http.StatusUnauthorized,
			wantHandled:  true,
			wantResponse: `{"message":"unauthorized"}`,
		},
		{
			name:        "passes a user actor through",
			actor:       &models.Actor{ID: "user-1", Type: models.ActorUser},
			wantNext:    true,
			wantHandled: false,
		},
		{
			name:        "passes a machine actor through",
			actor:       &models.Actor{ID: "key-1", Type: models.ActorMachine},
			wantNext:    true,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})

			req := newActorRequest(tt.actor)
			reqCtx, ok := models.GetRequestContext(req.Context())
			require.True(t, ok)

			RequireAuthenticated()(next).ServeHTTP(reqCtx.ResponseWriter, req)

			require.Equal(t, tt.wantNext, nextCalled)
			require.Equal(t, tt.wantHandled, reqCtx.Handled)
			require.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			if tt.wantResponse != "" {
				require.JSONEq(t, tt.wantResponse, string(reqCtx.ResponseBody))
				require.Equal(t, "application/json", reqCtx.ResponseHeaders.Get("Content-Type"))
			}
		})
	}
}

func TestRequireActor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		actor        *models.Actor
		allowed      []models.ActorType
		wantNext     bool
		wantStatus   int
		wantHandled  bool
		wantResponse string
	}{
		{
			name:         "rejects a request without an actor",
			actor:        nil,
			allowed:      []models.ActorType{models.ActorUser},
			wantNext:     false,
			wantStatus:   http.StatusUnauthorized,
			wantHandled:  true,
			wantResponse: `{"message":"unauthorized"}`,
		},
		{
			name:        "passes an allowed actor through",
			actor:       &models.Actor{ID: "user-1", Type: models.ActorUser},
			allowed:     []models.ActorType{models.ActorUser},
			wantNext:    true,
			wantHandled: false,
		},
		{
			name:        "passes an actor matching one of several allowed types",
			actor:       &models.Actor{ID: "key-1", Type: models.ActorMachine},
			allowed:     []models.ActorType{models.ActorUser, models.ActorMachine},
			wantNext:    true,
			wantHandled: false,
		},
		{
			name:         "rejects an actor with a disallowed type",
			actor:        &models.Actor{ID: "key-1", Type: models.ActorMachine},
			allowed:      []models.ActorType{models.ActorUser},
			wantNext:     false,
			wantStatus:   http.StatusForbidden,
			wantHandled:  true,
			wantResponse: `{"message":"only [user] actors are allowed for this action"}`,
		},
		{
			name:         "rejects every actor when no type is allowed",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			allowed:      nil,
			wantNext:     false,
			wantStatus:   http.StatusForbidden,
			wantHandled:  true,
			wantResponse: `{"message":"only [] actors are allowed for this action"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})

			req := newActorRequest(tt.actor)
			reqCtx, ok := models.GetRequestContext(req.Context())
			require.True(t, ok)

			RequireActor(tt.allowed...)(next).ServeHTTP(reqCtx.ResponseWriter, req)

			require.Equal(t, tt.wantNext, nextCalled)
			require.Equal(t, tt.wantHandled, reqCtx.Handled)
			require.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			if tt.wantResponse != "" {
				require.JSONEq(t, tt.wantResponse, string(reqCtx.ResponseBody))
			}
		})
	}
}

func TestRequirePublicOrUserActor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		actor        *models.Actor
		wantNext     bool
		wantStatus   int
		wantHandled  bool
		wantResponse string
	}{
		{
			name:        "passes an unauthenticated request through",
			actor:       nil,
			wantNext:    true,
			wantHandled: false,
		},
		{
			name:        "passes a user actor through",
			actor:       &models.Actor{ID: "user-1", Type: models.ActorUser},
			wantNext:    true,
			wantHandled: false,
		},
		{
			name:         "rejects a machine actor",
			actor:        &models.Actor{ID: "key-1", Type: models.ActorMachine},
			wantNext:     false,
			wantStatus:   http.StatusForbidden,
			wantHandled:  true,
			wantResponse: `{"message":"only public or user actors are allowed for this action"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})

			req := newActorRequest(tt.actor)
			reqCtx, ok := models.GetRequestContext(req.Context())
			require.True(t, ok)

			RequirePublicOrUserActor()(next).ServeHTTP(reqCtx.ResponseWriter, req)

			require.Equal(t, tt.wantNext, nextCalled)
			require.Equal(t, tt.wantHandled, reqCtx.Handled)
			require.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			if tt.wantResponse != "" {
				require.JSONEq(t, tt.wantResponse, string(reqCtx.ResponseBody))
			}
		})
	}
}
