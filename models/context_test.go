package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextKeyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  ContextKey
		want string
	}{
		{name: "user id", key: ContextUserID, want: "user_id"},
		{name: "auth actor", key: ContextAuthActor, want: "auth.actor"},
		{name: "request context", key: ContextRequestContext, want: "request_context"},
		{name: "custom key", key: ContextKey("custom"), want: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.key.String())
		})
	}
}

func TestRequestContextRoundTrip(t *testing.T) {
	t.Parallel()

	reqCtx := &RequestContext{Path: "/auth/me"}

	tests := []struct {
		name      string
		attach    func(context.Context, *RequestContext) context.Context
		wantFound bool
	}{
		{
			name:      "NewContextWithRequestContext attaches the context",
			attach:    NewContextWithRequestContext,
			wantFound: true,
		},
		{
			name:      "SetRequestContext attaches the context",
			attach:    SetRequestContext,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.attach(context.Background(), reqCtx)

			got, ok := GetRequestContext(ctx)

			require.Equal(t, tt.wantFound, ok)
			require.Same(t, reqCtx, got)
		})
	}
}

func TestGetRequestContextMissingValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "empty context", ctx: context.Background()},
		{
			name: "value of the wrong type",
			ctx:  context.WithValue(context.Background(), ContextRequestContext, "not-a-request-context"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := GetRequestContext(tt.ctx)

			require.False(t, ok)
			require.Nil(t, got)
		})
	}
}

func TestSetResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		headers     http.Header
		wantHeaders http.Header
	}{
		{
			name:        "nil headers clear the response headers",
			headers:     nil,
			wantHeaders: nil,
		},
		{
			name:        "headers are copied",
			headers:     http.Header{"X-Test": {"one", "two"}, "Content-Type": {"application/json"}},
			wantHeaders: http.Header{"X-Test": {"one", "two"}, "Content-Type": {"application/json"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &RequestContext{}
			body := []byte(`{"ok":true}`)

			reqCtx.SetResponse(http.StatusCreated, tt.headers, body)

			require.Equal(t, http.StatusCreated, reqCtx.ResponseStatus)
			require.Equal(t, tt.wantHeaders, reqCtx.ResponseHeaders)
			require.Equal(t, body, reqCtx.ResponseBody)
			require.True(t, reqCtx.ResponseReady)

			if tt.headers != nil {
				// The captured headers must not alias the caller's slice.
				tt.headers["X-Test"][0] = "mutated"
				require.Equal(t, "one", reqCtx.ResponseHeaders.Get("X-Test"))
			}
		})
	}
}

func TestSetJSONResponse(t *testing.T) {
	t.Parallel()

	reqCtx := &RequestContext{}

	reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "unauthorized"})

	require.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
	require.True(t, reqCtx.ResponseReady)
	require.Equal(t, "application/json", reqCtx.ResponseHeaders.Get("Content-Type"))
	require.JSONEq(t, `{"message":"unauthorized"}`, string(reqCtx.ResponseBody))
}

func TestSetJSONResponseFallsBackWhenMarshalingFails(t *testing.T) {
	t.Parallel()

	reqCtx := &RequestContext{}

	reqCtx.SetJSONResponse(http.StatusOK, make(chan int))

	require.Equal(t, http.StatusInternalServerError, reqCtx.ResponseStatus)
	require.True(t, reqCtx.ResponseReady)
	require.Equal(t, "text/plain", reqCtx.ResponseHeaders.Get("Content-Type"))
	require.Equal(t, "Internal Server Error", string(reqCtx.ResponseBody))
}

func TestSetActorInContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		actor         *Actor
		wantNilActor  bool
		wantUserID    string
		wantUserIDSet bool
	}{
		{
			name:         "nil actor clears the actor and leaves user id unset",
			actor:        nil,
			wantNilActor: true,
		},
		{
			name:          "actor with an ID populates both context keys",
			actor:         &Actor{ID: "user-1", Type: ActorUser},
			wantUserID:    "user-1",
			wantUserIDSet: true,
		},
		{
			name:  "actor without an ID does not populate the user id key",
			actor: &Actor{Type: ActorMachine},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			reqCtx := &RequestContext{Request: req}

			reqCtx.SetActorInContext(tt.actor)

			if tt.wantNilActor {
				require.Nil(t, reqCtx.Actor)
				return
			}

			require.Same(t, tt.actor, reqCtx.Actor)
			require.NotNil(t, reqCtx.Actor.Scopes)
			require.NotNil(t, reqCtx.Actor.Claims)

			fromCtx, ok := reqCtx.Request.Context().Value(ContextAuthActor).(*Actor)
			require.True(t, ok)
			require.Same(t, tt.actor, fromCtx)

			userID, set := reqCtx.Request.Context().Value(ContextUserID).(string)
			require.Equal(t, tt.wantUserIDSet, reqCtx.Request.Context().Value(ContextUserID) != nil)
			if tt.wantUserIDSet {
				require.Equal(t, tt.wantUserID, userID)
				require.True(t, set)
			}
		})
	}
}

func TestSetActorInContextPreservesExistingScopesAndClaims(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqCtx := &RequestContext{Request: req}
	actor := &Actor{
		ID:     "user-1",
		Type:   ActorUser,
		Scopes: []string{"users:read"},
		Claims: map[string]any{"role": "admin"},
	}

	reqCtx.SetActorInContext(actor)

	require.Equal(t, []string{"users:read"}, actor.Scopes)
	require.Equal(t, map[string]any{"role": "admin"}, actor.Claims)
}

func TestGetActorFromContext(t *testing.T) {
	t.Parallel()

	actor := &Actor{ID: "user-1", Type: ActorUser}

	tests := []struct {
		name      string
		ctx       context.Context
		wantActor *Actor
		wantOK    bool
	}{
		{
			name:      "returns the actor stored directly in the context",
			ctx:       context.WithValue(context.Background(), ContextAuthActor, actor),
			wantActor: actor,
			wantOK:    true,
		},
		{
			name: "falls back to the request context actor",
			ctx: NewContextWithRequestContext(
				context.Background(),
				&RequestContext{Actor: actor},
			),
			wantActor: actor,
			wantOK:    true,
		},
		{
			name: "falls back when the context value has the wrong type",
			ctx: NewContextWithRequestContext(
				context.WithValue(context.Background(), ContextAuthActor, "not-an-actor"),
				&RequestContext{Actor: actor},
			),
			wantActor: actor,
			wantOK:    true,
		},
		{
			name:   "returns false when there is no actor",
			ctx:    context.Background(),
			wantOK: false,
		},
		{
			name: "returns false when the request context has no actor",
			ctx: NewContextWithRequestContext(
				context.Background(),
				&RequestContext{},
			),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := GetActorFromContext(tt.ctx)

			require.Equal(t, tt.wantOK, ok)
			require.Same(t, tt.wantActor, got)
		})
	}
}

func TestGetActorFromRequest(t *testing.T) {
	t.Parallel()

	actor := &Actor{ID: "user-1", Type: ActorUser}
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqCtx := &RequestContext{Request: req}
	reqCtx.SetActorInContext(actor)

	got, ok := GetActorFromRequest(reqCtx.Request)

	require.True(t, ok)
	require.Same(t, actor, got)
}
