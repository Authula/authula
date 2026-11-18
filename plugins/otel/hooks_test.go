package otel

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/otel/types"
)

type fixture struct {
	plugin   *OTelPlugin
	recorder *tracetest.SpanRecorder
	reader   *sdkmetric.ManualReader
}

func newFixture(t *testing.T, config types.OTelPluginConfig) *fixture {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	plugin := &OTelPlugin{
		pluginConfig:   config,
		propagator:     propagation.TraceContext{},
		tracerProvider: tracerProvider,
		tracer:         tracerProvider.Tracer("test"),
	}
	if err := plugin.bindMeter(meterProvider.Meter("test")); err != nil {
		t.Fatalf("bindMeter() error = %v", err)
	}
	plugin.meterProvider = meterProvider

	t.Cleanup(func() {
		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
	})

	return &fixture{plugin: plugin, recorder: recorder, reader: reader}
}

func newRequestContext(method, path, pattern string) *models.RequestContext {
	req := httptest.NewRequest(method, path, nil)
	return &models.RequestContext{
		Request: req,
		Method:  method,
		Path:    path,
		Values:  make(map[string]any),
		Route:   &models.Route{Method: method, Path: path, Pattern: pattern},
	}
}

func findAttribute(span sdktrace.ReadOnlySpan, key attribute.Key) (attribute.Value, bool) {
	for _, kv := range span.Attributes() {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestHookOnRequestPropagatesTraceContext(t *testing.T) {
	t.Parallel()

	f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
	reqCtx := newRequestContext(http.MethodGet, "/api/auth/me", "GET:/api/auth/me")
	reqCtx.Request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	if err := f.plugin.onRequest()(reqCtx); err != nil {
		t.Fatalf("onRequest() error = %v", err)
	}

	span := trace.SpanFromContext(reqCtx.Request.Context())
	if !span.SpanContext().IsValid() {
		t.Fatal("handler context does not carry a valid span")
	}

	want, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("bad trace ID: %v", err)
	}
	if span.SpanContext().TraceID() != want {
		t.Fatalf("trace ID = %s, want %s", span.SpanContext().TraceID(), want)
	}

	if err := f.plugin.onComplete()(reqCtx); err != nil {
		t.Fatalf("onComplete() error = %v", err)
	}

	ended := f.recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Name() != "GET /api/auth/me" {
		t.Fatalf("span name = %q, want %q", ended[0].Name(), "GET /api/auth/me")
	}
	if ended[0].SpanKind() != trace.SpanKindServer {
		t.Fatalf("span kind = %v, want server", ended[0].SpanKind())
	}
}

func TestFinalizeSpanAttributes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		status         int
		actor          *models.Actor
		wantStatusAttr int
		wantErrorType  bool
		wantSpanError  bool
	}{
		{name: "200 success", status: http.StatusOK, wantStatusAttr: 200},
		{name: "404 client error", status: http.StatusNotFound, wantStatusAttr: 404},
		{name: "500 server error", status: http.StatusInternalServerError, wantStatusAttr: 500, wantErrorType: true, wantSpanError: true},
		{
			name:           "actor identity",
			status:         http.StatusOK,
			actor:          &models.Actor{Type: models.ActorUser, ID: "user-123"},
			wantStatusAttr: 200,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
			reqCtx := newRequestContext(http.MethodGet, "/api/auth/me", "GET:/api/auth/me")
			reqCtx.ClientIP = "127.0.0.1"
			reqCtx.ResponseStatus = tc.status
			reqCtx.Actor = tc.actor
			reqCtx.Request.Header.Set("User-Agent", "authula-test/1.0")

			if err := f.plugin.onRequest()(reqCtx); err != nil {
				t.Fatalf("onRequest() error = %v", err)
			}
			if err := f.plugin.onComplete()(reqCtx); err != nil {
				t.Fatalf("onComplete() error = %v", err)
			}

			ended := f.recorder.Ended()
			if len(ended) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(ended))
			}
			span := ended[0]

			if v, ok := findAttribute(span, attribute.Key("http.request.method")); !ok || v.AsString() != http.MethodGet {
				t.Errorf("http.request.method = %v (ok=%v), want GET", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("http.route")); !ok || v.AsString() != "/api/auth/me" {
				t.Errorf("http.route = %q (ok=%v), want /api/auth/me", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("url.path")); !ok || v.AsString() != "/api/auth/me" {
				t.Errorf("url.path = %q (ok=%v), want /api/auth/me", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("url.scheme")); !ok || v.AsString() != "http" {
				t.Errorf("url.scheme = %q (ok=%v), want http", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("client.address")); !ok || v.AsString() != "127.0.0.1" {
				t.Errorf("client.address = %q (ok=%v), want 127.0.0.1", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("http.response.status_code")); !ok || v.AsInt64() != int64(tc.wantStatusAttr) {
				t.Errorf("http.response.status_code = %d (ok=%v), want %d", v.AsInt64(), ok, tc.wantStatusAttr)
			}
			if v, ok := findAttribute(span, attribute.Key("server.address")); !ok || v.AsString() != "example.com" {
				t.Errorf("server.address = %q (ok=%v), want example.com", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("server.port")); !ok || v.AsInt64() != 80 {
				t.Errorf("server.port = %d (ok=%v), want 80", v.AsInt64(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("network.protocol.version")); !ok || v.AsString() != "1.1" {
				t.Errorf("network.protocol.version = %q (ok=%v), want 1.1", v.AsString(), ok)
			}
			if v, ok := findAttribute(span, attribute.Key("user_agent.original")); !ok || v.AsString() != "authula-test/1.0" {
				t.Errorf("user_agent.original = %q (ok=%v), want authula-test/1.0", v.AsString(), ok)
			}
			if _, ok := findAttribute(span, attribute.Key("url.query")); ok {
				t.Error("url.query should not be recorded")
			}

			_, hasErrorType := findAttribute(span, attribute.Key(errorTypeKey))
			if hasErrorType != tc.wantErrorType {
				t.Errorf("error.type present = %v, want %v", hasErrorType, tc.wantErrorType)
			}

			wantCode := codes.Unset
			if tc.wantSpanError {
				wantCode = codes.Error
			}
			if span.Status().Code != wantCode {
				t.Errorf("span status = %v, want %v", span.Status().Code, wantCode)
			}

			if tc.actor != nil {
				if v, ok := findAttribute(span, attribute.Key("authula.actor.type")); !ok || v.AsString() != string(tc.actor.Type) {
					t.Errorf("authula.actor.type = %q (ok=%v), want %q", v.AsString(), ok, tc.actor.Type)
				}
				if v, ok := findAttribute(span, attribute.Key("authula.actor.id")); !ok || v.AsString() != tc.actor.ID {
					t.Errorf("authula.actor.id = %q (ok=%v), want %q", v.AsString(), ok, tc.actor.ID)
				}
			}
		})
	}
}

func TestFinalizeUnmatchedRoute(t *testing.T) {
	t.Parallel()

	f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
	reqCtx := newRequestContext(http.MethodPost, "/api/auth/unknown", "")
	reqCtx.ResponseStatus = http.StatusNotFound

	if err := f.plugin.onRequest()(reqCtx); err != nil {
		t.Fatalf("onRequest() error = %v", err)
	}
	if err := f.plugin.onComplete()(reqCtx); err != nil {
		t.Fatalf("onComplete() error = %v", err)
	}

	span := f.recorder.Ended()[0]
	if span.Name() != "POST" {
		t.Errorf("span name = %q, want %q", span.Name(), "POST")
	}
	if v, ok := findAttribute(span, attribute.Key("http.route")); !ok || v.AsString() != routeUnmatched {
		t.Errorf("http.route = %q (ok=%v), want %q", v.AsString(), ok, routeUnmatched)
	}
	if v, ok := findAttribute(span, attribute.Key("url.path")); !ok || v.AsString() != "/api/auth/unknown" {
		t.Errorf("url.path = %q, want /api/auth/unknown", v.AsString())
	}
}

func TestFinalizeUnknownMethodMappedToOther(t *testing.T) {
	t.Parallel()

	f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
	reqCtx := newRequestContext("BREW", "/api/auth/coffee", "")

	if err := f.plugin.onRequest()(reqCtx); err != nil {
		t.Fatalf("onRequest() error = %v", err)
	}
	if err := f.plugin.onComplete()(reqCtx); err != nil {
		t.Fatalf("onComplete() error = %v", err)
	}

	span := f.recorder.Ended()[0]
	if v, ok := findAttribute(span, attribute.Key("http.request.method")); !ok || v.AsString() != "_OTHER" {
		t.Errorf("http.request.method = %q (ok=%v), want _OTHER", v.AsString(), ok)
	}
	if v, ok := findAttribute(span, attribute.Key("http.request.method_original")); !ok || v.AsString() != "BREW" {
		t.Errorf("http.request.method_original = %q (ok=%v), want BREW", v.AsString(), ok)
	}
}

func TestResolveSchemeAndClient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		trustProxy     bool
		tls            bool
		forwardedProto string
		forwardedFor   string
		wantScheme     string
		wantClient     string
	}{
		{name: "plain http", wantScheme: "http", wantClient: "127.0.0.1"},
		{name: "tls is https", tls: true, wantScheme: "https", wantClient: "127.0.0.1"},
		{
			name:           "spoofed headers ignored without trust",
			forwardedProto: "https",
			forwardedFor:   "1.2.3.4",
			wantScheme:     "http",
			wantClient:     "127.0.0.1",
		},
		{
			name:           "headers honored with trust",
			trustProxy:     true,
			forwardedProto: "https",
			forwardedFor:   "1.2.3.4, 5.6.7.8",
			wantScheme:     "https",
			wantClient:     "1.2.3.4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := newRequestContext(http.MethodGet, "/", "")
			reqCtx.ClientIP = "127.0.0.1"
			if tc.tls {
				reqCtx.Request.TLS = &tls.ConnectionState{}
			}
			if tc.forwardedProto != "" {
				reqCtx.Request.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
			}
			if tc.forwardedFor != "" {
				reqCtx.Request.Header.Set("X-Forwarded-For", tc.forwardedFor)
			}

			scheme, client := resolveSchemeAndClient(reqCtx, tc.trustProxy)
			if scheme != tc.wantScheme {
				t.Errorf("scheme = %q, want %q", scheme, tc.wantScheme)
			}
			if client != tc.wantClient {
				t.Errorf("client = %q, want %q", client, tc.wantClient)
			}
		})
	}
}

func TestResolveServer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		host       string
		scheme     string
		trustProxy bool
		forwarded  string
		wantHost   string
		wantPort   int
	}{
		{name: "host with port", host: "api.example.com:8443", scheme: "http", wantHost: "api.example.com", wantPort: 8443},
		{name: "host without port", host: "api.example.com", scheme: "http", wantHost: "api.example.com", wantPort: 80},
		{name: "https default port", host: "api.example.com", scheme: "https", wantHost: "api.example.com", wantPort: 443},
		{name: "ipv6 with port", host: "[::1]:8080", scheme: "http", wantHost: "::1", wantPort: 8080},
		{name: "forwarded host honored", host: "internal:8080", scheme: "http", trustProxy: true, forwarded: "public.example.com:9443", wantHost: "public.example.com", wantPort: 9443},
		{name: "forwarded host ignored without trust", host: "internal:8080", scheme: "http", forwarded: "public.example.com", wantHost: "internal", wantPort: 8080},
		{name: "empty host", scheme: "http", wantHost: "", wantPort: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := newRequestContext(http.MethodGet, "/", "")
			reqCtx.Request.Host = tc.host
			if tc.forwarded != "" {
				reqCtx.Request.Header.Set("X-Forwarded-Host", tc.forwarded)
			}

			host, port := resolveServer(reqCtx, tc.scheme, tc.trustProxy)
			if host != tc.wantHost {
				t.Errorf("host = %q, want %q", host, tc.wantHost)
			}
			if port != tc.wantPort {
				t.Errorf("port = %d, want %d", port, tc.wantPort)
			}
		})
	}
}

func TestProtocolVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proto string
		want  string
	}{
		{name: "http 1.1", proto: "HTTP/1.1", want: "1.1"},
		{name: "http 1.0", proto: "HTTP/1.0", want: "1.0"},
		{name: "http 2.0", proto: "HTTP/2.0", want: "2"},
		{name: "http 3.0", proto: "HTTP/3.0", want: "3"},
		{name: "empty proto", proto: "", want: ""},
		{name: "garbage proto", proto: "garbage", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := newRequestContext(http.MethodGet, "/", "")
			reqCtx.Request.Proto = tc.proto

			if got := protocolVersion(reqCtx); got != tc.want {
				t.Errorf("protocolVersion(%q) = %q, want %q", tc.proto, got, tc.want)
			}
		})
	}
}

func TestFinalizeRecordsDurationMetric(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		method        string
		status        int
		wantMetric    bool
		wantErrorType bool
	}{
		{name: "request recorded", method: http.MethodGet, status: http.StatusOK, wantMetric: true},
		{name: "5xx records error.type", method: http.MethodPost, status: http.StatusInternalServerError, wantMetric: true, wantErrorType: true},
		{name: "preflight not recorded", method: http.MethodOptions, status: http.StatusOK, wantMetric: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
			reqCtx := newRequestContext(tc.method, "/api/auth/me", "GET:/api/auth/me")
			reqCtx.ResponseStatus = tc.status

			if err := f.plugin.onRequest()(reqCtx); err != nil {
				t.Fatalf("onRequest() error = %v", err)
			}
			if err := f.plugin.onComplete()(reqCtx); err != nil {
				t.Fatalf("onComplete() error = %v", err)
			}

			metric, found := collectMetric(t, f.reader)
			if found != tc.wantMetric {
				t.Fatalf("metric found = %v, want %v", found, tc.wantMetric)
			}
			if !tc.wantMetric {
				return
			}

			if metric.Name != durationInstrumentName {
				t.Errorf("metric name = %q, want %q", metric.Name, durationInstrumentName)
			}
			if metric.Unit != "s" {
				t.Errorf("metric unit = %q, want s", metric.Unit)
			}

			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("metric data type = %T, want histogram", metric.Data)
			}
			if len(histogram.DataPoints) != 1 {
				t.Fatalf("data points = %d, want 1", len(histogram.DataPoints))
			}

			point := histogram.DataPoints[0]
			if point.Count != 1 {
				t.Errorf("count = %d, want 1", point.Count)
			}
			if point.Sum <= 0 {
				t.Errorf("sum = %v, want > 0", point.Sum)
			}

			if v, ok := point.Attributes.Value(attribute.Key("http.response.status_code")); !ok || v.AsInt64() != int64(tc.status) {
				t.Errorf("http.response.status_code = %d (ok=%v), want %d", v.AsInt64(), ok, tc.status)
			}
			if v, ok := point.Attributes.Value(attribute.Key("url.scheme")); !ok || v.AsString() != "http" {
				t.Errorf("url.scheme = %q (ok=%v), want http", v.AsString(), ok)
			}
			_, hasErrorType := point.Attributes.Value(attribute.Key(errorTypeKey))
			if hasErrorType != tc.wantErrorType {
				t.Errorf("error.type present = %v, want %v", hasErrorType, tc.wantErrorType)
			}
		})
	}
}

func TestFinalizeIdempotentAndShortCircuit(t *testing.T) {
	t.Parallel()

	f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})
	reqCtx := newRequestContext(http.MethodGet, "/api/auth/me", "GET:/api/auth/me")
	reqCtx.ResponseWriter = httptest.NewRecorder()
	reqCtx.ResponseStatus = http.StatusTooManyRequests

	if err := f.plugin.onRequest()(reqCtx); err != nil {
		t.Fatalf("onRequest() error = %v", err)
	}
	// Simulate a short-circuit: no handler ran, completion still finalizes.
	if err := f.plugin.onComplete()(reqCtx); err != nil {
		t.Fatalf("onComplete() error = %v", err)
	}
	// A second completion must not end the span again.
	if err := f.plugin.onComplete()(reqCtx); err != nil {
		t.Fatalf("second onComplete() error = %v", err)
	}

	if got := len(f.recorder.Ended()); got != 1 {
		t.Fatalf("ended spans = %d, want 1", got)
	}
}

func TestHooksNeverReturnError(t *testing.T) {
	t.Parallel()

	f := newFixture(t, types.OTelPluginConfig{Enabled: true, TracesEnabled: new(true), MetricsEnabled: new(true)})

	// Empty request context: no route, no values.
	empty := &models.RequestContext{Method: http.MethodGet, Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	if err := f.plugin.onRequest()(empty); err != nil {
		t.Fatalf("onRequest() on empty context error = %v", err)
	}
	if err := f.plugin.onComplete()(empty); err != nil {
		t.Fatalf("onComplete() on empty context error = %v", err)
	}
}

func collectMetric(t *testing.T, reader sdkmetric.Reader) (metricdata.Metrics, bool) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &resourceMetrics); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == durationInstrumentName {
				return metric, true
			}
		}
	}

	return metricdata.Metrics{}, false
}

// statusResponseWriter exposes the status code through models.ResponseStatusProvider.
type statusResponseWriter struct {
	header http.Header
	status int
}

func (w *statusResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *statusResponseWriter) Write(body []byte) (int, error) { return len(body), nil }
func (w *statusResponseWriter) WriteHeader(status int)         { w.status = status }
func (w *statusResponseWriter) StatusCode() int                { return w.status }

func TestResolveStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		responseStatus int
		writer         http.ResponseWriter
		want           int
	}{
		{
			name:           "explicit response status wins",
			responseStatus: http.StatusTeapot,
			writer:         &statusResponseWriter{status: http.StatusOK},
			want:           http.StatusTeapot,
		},
		{
			name:   "response writer status is used",
			writer: &statusResponseWriter{status: http.StatusTooManyRequests},
			want:   http.StatusTooManyRequests,
		},
		{
			name:   "zero writer status falls back to 200",
			writer: &statusResponseWriter{},
			want:   http.StatusOK,
		},
		{
			name:   "writer without status provider falls back to 200",
			writer: httptest.NewRecorder(),
			want:   http.StatusOK,
		},
		{
			name: "nil writer falls back to 200",
			want: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{
				ResponseStatus: tc.responseStatus,
				ResponseWriter: tc.writer,
			}

			assert.Equal(t, tc.want, resolveStatus(reqCtx))
		})
	}
}

func TestRequestContext(t *testing.T) {
	t.Parallel()

	type contextKey string

	tests := []struct {
		name   string
		reqCtx *models.RequestContext
		check  func(t *testing.T, ctx context.Context)
	}{
		{
			name: "uses the request context when present",
			reqCtx: &models.RequestContext{
				Request: httptest.NewRequest(http.MethodGet, "/", nil).
					WithContext(context.WithValue(context.Background(), contextKey("marker"), "present")),
			},
			check: func(t *testing.T, ctx context.Context) {
				assert.Equal(t, "present", ctx.Value(contextKey("marker")))
			},
		},
		{
			name:   "falls back to background when request is nil",
			reqCtx: &models.RequestContext{},
			check: func(t *testing.T, ctx context.Context) {
				require.NotNil(t, ctx)
				assert.Nil(t, ctx.Value(contextKey("marker")))
				assert.Nil(t, ctx.Done())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.check(t, requestContext(tc.reqCtx))
		})
	}
}

func TestRequestMethodAttr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method string
		want   string
	}{
		{name: "connect", method: http.MethodConnect, want: "CONNECT"},
		{name: "head", method: http.MethodHead, want: "HEAD"},
		{name: "options", method: http.MethodOptions, want: "OPTIONS"},
		{name: "get", method: http.MethodGet, want: "GET"},
		{name: "post", method: http.MethodPost, want: "POST"},
		{name: "put", method: http.MethodPut, want: "PUT"},
		{name: "patch", method: http.MethodPatch, want: "PATCH"},
		{name: "delete", method: http.MethodDelete, want: "DELETE"},
		{name: "trace", method: http.MethodTrace, want: "TRACE"},
		{name: "unknown method", method: "BREW", want: "_OTHER"},
		{name: "empty method", method: "", want: "_OTHER"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			attr := requestMethodAttr(tc.method)

			assert.Equal(t, attribute.Key("http.request.method"), attr.Key)
			assert.Equal(t, tc.want, attr.Value.AsString())
		})
	}
}

func TestRoutePattern(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		route *models.Route
		want  string
	}{
		{name: "nil route", route: nil, want: ""},
		{name: "empty pattern", route: &models.Route{Pattern: ""}, want: ""},
		{name: "method prefix stripped", route: &models.Route{Pattern: "GET:/api/auth/me"}, want: "/api/auth/me"},
		{name: "plain pattern kept", route: &models.Route{Pattern: "/api/auth/me"}, want: "/api/auth/me"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{Route: tc.route}

			assert.Equal(t, tc.want, routePattern(reqCtx))
		})
	}
}

func TestFinalizeDropsInvalidSpanState(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
	}{
		{name: "wrong value type", value: "not-a-span-state"},
		{name: "typed nil state", value: (*spanState)(nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := &OTelPlugin{}
			reqCtx := &models.RequestContext{
				Values: map[string]any{spanStateKey: tc.value},
			}

			plugin.finalize(reqCtx)

			_, exists := reqCtx.Values[spanStateKey]
			assert.False(t, exists)
		})
	}
}

func TestOnRequestWithoutRequest(t *testing.T) {
	t.Parallel()

	plugin := &OTelPlugin{}
	reqCtx := &models.RequestContext{Method: http.MethodGet}

	require.NoError(t, plugin.onRequest()(reqCtx))

	state, ok := reqCtx.Values[spanStateKey].(*spanState)
	require.True(t, ok)
	require.NotNil(t, state)
	assert.Nil(t, state.span)
	assert.False(t, state.start.IsZero())
}

func TestHelpersWithoutRequest(t *testing.T) {
	t.Parallel()

	reqCtx := &models.RequestContext{ClientIP: "127.0.0.1"}

	scheme, client := resolveSchemeAndClient(reqCtx, true)
	assert.Equal(t, "http", scheme)
	assert.Equal(t, "127.0.0.1", client)

	host, port := resolveServer(reqCtx, "https", true)
	assert.Empty(t, host)
	assert.Zero(t, port)

	assert.Empty(t, protocolVersion(reqCtx))
	assert.NotNil(t, requestContext(reqCtx))
}
