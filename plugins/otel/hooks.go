package otel

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Authula/authula/models"
)

const (
	// spanStateKey is the key under reqCtx.Values holding the active span state.
	spanStateKey = "otel.span.state"
	// durationInstrumentName is the semantic-convention HTTP server duration
	// histogram name.
	durationInstrumentName = "http.server.request.duration"
	// routeUnmatched is the fixed, low-cardinality label used when no route was
	// matched.
	routeUnmatched = "unmatched"
	// errorTypeKey is the semantic-convention attribute key for the error class.
	errorTypeKey = "error.type"
)

// durationBuckets are the semantic-convention recommended histogram boundaries
// for HTTP server request duration, in seconds.
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}

// spanState carries the per-request span and its start time between the
// on-request and on-complete hooks.
type spanState struct {
	span  trace.Span
	start time.Time
}

// buildHooks wires the request and completion hooks. When both traces and
// metrics are off the plugin contributes no hooks at all.
func (p *OTelPlugin) buildHooks() []models.Hook {
	if !p.pluginConfig.TracesOn() && !p.pluginConfig.MetricsOn() {
		return nil
	}

	return []models.Hook{
		{
			Stage:       models.HookOnRequest,
			Handler:     p.onRequest(),
			Order:       -100,
			RunOnBypass: true,
		},
		{
			Stage:   models.HookOnComplete,
			Handler: p.onComplete(),
		},
	}
}

func (p *OTelPlugin) onRequest() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		start := time.Now()

		if reqCtx.Values == nil {
			reqCtx.Values = make(map[string]any)
		}

		if reqCtx.Request == nil {
			reqCtx.Values[spanStateKey] = &spanState{start: start}
			return nil
		}

		if !p.pluginConfig.TracesOn() {
			reqCtx.Values[spanStateKey] = &spanState{start: start}
			return nil
		}

		ctx := p.propagator.Extract(reqCtx.Request.Context(), propagation.HeaderCarrier(reqCtx.Request.Header))

		route := routePattern(reqCtx)
		spanName := reqCtx.Method
		if route != "" {
			spanName = reqCtx.Method + " " + route
		}

		ctx, span := p.tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithTimestamp(start),
		)

		reqCtx.Request = reqCtx.Request.WithContext(ctx)
		reqCtx.Values[spanStateKey] = &spanState{span: span, start: start}

		return nil
	}
}

func (p *OTelPlugin) onComplete() models.HookHandler {
	return func(reqCtx *models.RequestContext) error {
		p.finalize(reqCtx)
		return nil
	}
}

// finalize ends the span and records the request metric. It is idempotent: a
// missing span state (already finalized, or never started) is a no-op.
func (p *OTelPlugin) finalize(reqCtx *models.RequestContext) {
	raw, exists := reqCtx.Values[spanStateKey]
	if !exists {
		return
	}

	state, ok := raw.(*spanState)
	if !ok || state == nil {
		delete(reqCtx.Values, spanStateKey)
		return
	}

	status := resolveStatus(reqCtx)
	route := routePattern(reqCtx)
	scheme, clientAddress := resolveSchemeAndClient(reqCtx, p.pluginConfig.TrustProxy)
	ctx := requestContext(reqCtx)

	if p.pluginConfig.TracesOn() && state.span != nil {
		p.recordSpan(state.span, reqCtx, status, route, scheme, clientAddress)
	}

	if p.pluginConfig.MetricsOn() && p.duration != nil && reqCtx.Method != http.MethodOptions {
		attrs := []attribute.KeyValue{
			requestMethodAttr(reqCtx.Method),
			semconv.HTTPRoute(routeLabel(route)),
			semconv.HTTPResponseStatusCode(status),
			semconv.URLScheme(scheme),
		}
		if status >= http.StatusInternalServerError {
			attrs = append(attrs, attribute.String(errorTypeKey, strconv.Itoa(status)))
		}
		p.duration.Record(ctx, time.Since(state.start).Seconds(), metric.WithAttributes(attrs...))
	}

	if state.span != nil {
		state.span.End()
	}
	delete(reqCtx.Values, spanStateKey)
}

// recordSpan attaches the HTTP metadata and, for 5xx responses only, the error
// status. Per the stable HTTP semantic conventions a 4xx server span is not an
// error, so its status stays unset. url.query is deliberately omitted because
// Authula carries credentials such as email-verification and magic-link tokens
// in the query string.
func (p *OTelPlugin) recordSpan(span trace.Span, reqCtx *models.RequestContext, status int, route, scheme, clientAddress string) {
	attrs := []attribute.KeyValue{
		requestMethodAttr(reqCtx.Method),
		semconv.HTTPRoute(routeLabel(route)),
		semconv.URLPath(reqCtx.Path),
		semconv.URLScheme(scheme),
		semconv.ClientAddress(clientAddress),
		semconv.HTTPResponseStatusCode(status),
	}

	if methodOriginal, ok := methodOriginalAttr(reqCtx.Method); ok {
		attrs = append(attrs, methodOriginal)
	}
	if host, port := resolveServer(reqCtx, scheme, p.pluginConfig.TrustProxy); host != "" {
		attrs = append(attrs, semconv.ServerAddress(host), semconv.ServerPort(port))
	}
	if version := protocolVersion(reqCtx); version != "" {
		attrs = append(attrs, semconv.NetworkProtocolVersion(version))
	}
	if reqCtx.Request != nil {
		if userAgent := reqCtx.Request.UserAgent(); userAgent != "" {
			attrs = append(attrs, semconv.UserAgentOriginal(userAgent))
		}
	}

	if reqCtx.Actor != nil {
		attrs = append(attrs,
			attribute.String("authula.actor.type", reqCtx.Actor.Type.ToString()),
			attribute.String("authula.actor.id", reqCtx.Actor.ID),
		)
	}

	span.SetAttributes(attrs...)

	if status >= http.StatusInternalServerError {
		span.SetStatus(codes.Error, http.StatusText(status))
		span.SetAttributes(attribute.String(errorTypeKey, strconv.Itoa(status)))
	}
}

// routePattern returns the matched route with its leading "METHOD:" prefix
// stripped, or an empty string when no route was matched.
func routePattern(reqCtx *models.RequestContext) string {
	if reqCtx.Route == nil {
		return ""
	}
	pattern := reqCtx.Route.Pattern
	if pattern == "" {
		return ""
	}
	if idx := strings.IndexByte(pattern, ':'); idx >= 0 {
		return pattern[idx+1:]
	}
	return pattern
}

func routeLabel(route string) string {
	if route == "" {
		return routeUnmatched
	}
	return route
}

// resolveStatus resolves the final HTTP status from the captured response, the
// response writer, or defaults to 200.
func resolveStatus(reqCtx *models.RequestContext) int {
	if reqCtx.ResponseStatus != 0 {
		return reqCtx.ResponseStatus
	}
	if provider, ok := reqCtx.ResponseWriter.(models.ResponseStatusProvider); ok {
		if code := provider.StatusCode(); code != 0 {
			return code
		}
	}
	return http.StatusOK
}

// resolveSchemeAndClient derives the request scheme and client address. With
// trust_proxy enabled the X-Forwarded-Proto/X-Forwarded-For headers take
// precedence; otherwise they are ignored.
func resolveSchemeAndClient(reqCtx *models.RequestContext, trustProxy bool) (string, string) {
	scheme := "http"
	clientAddress := reqCtx.ClientIP

	if reqCtx.Request == nil {
		return scheme, clientAddress
	}

	if reqCtx.Request.TLS != nil {
		scheme = "https"
	}

	if !trustProxy {
		return scheme, clientAddress
	}

	if proto := strings.TrimSpace(reqCtx.Request.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = strings.ToLower(proto)
	}
	if forwardedFor := reqCtx.Request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if first := strings.TrimSpace(strings.Split(forwardedFor, ",")[0]); first != "" {
			clientAddress = first
		}
	}

	return scheme, clientAddress
}

// resolveServer derives server.address and server.port from the request
// authority, preferring X-Forwarded-Host when proxy headers are trusted. If the
// authority carries no port, the scheme default is used.
func resolveServer(reqCtx *models.RequestContext, scheme string, trustProxy bool) (string, int) {
	if reqCtx.Request == nil {
		return "", 0
	}

	authority := reqCtx.Request.Host
	if trustProxy {
		if forwarded := strings.TrimSpace(reqCtx.Request.Header.Get("X-Forwarded-Host")); forwarded != "" {
			authority = forwarded
		}
	}
	if authority == "" {
		return "", 0
	}

	host, port := authority, defaultPort(scheme)
	if h, p, err := net.SplitHostPort(authority); err == nil {
		host = h
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	return host, port
}

// defaultPort returns the default port for the scheme.
func defaultPort(scheme string) int {
	if scheme == "https" {
		return 443
	}
	return 80
}

// protocolVersion converts the Go request protocol ("HTTP/1.1") to the
// semantic-convention form ("1.1"). It returns an empty string when the
// version is unknown.
func protocolVersion(reqCtx *models.RequestContext) string {
	if reqCtx.Request == nil {
		return ""
	}

	version, ok := strings.CutPrefix(reqCtx.Request.Proto, "HTTP/")
	if !ok {
		return ""
	}
	switch version {
	case "2.0":
		return "2"
	case "3.0":
		return "3"
	default:
		return version
	}
}

// methodOriginalAttr returns the http.request.method_original attribute when
// the method was mapped to _OTHER, as the semantic conventions require.
func methodOriginalAttr(method string) (attribute.KeyValue, bool) {
	if requestMethodAttr(method).Value.AsString() != "_OTHER" {
		return attribute.KeyValue{}, false
	}
	return semconv.HTTPRequestMethodOriginal(method), true
}

func requestContext(reqCtx *models.RequestContext) context.Context {
	if reqCtx.Request != nil {
		return reqCtx.Request.Context()
	}
	return context.Background()
}

// requestMethodAttr maps the HTTP method to its semantic-convention attribute,
// falling back to _OTHER for unknown methods.
func requestMethodAttr(method string) attribute.KeyValue {
	switch method {
	case http.MethodConnect:
		return semconv.HTTPRequestMethodConnect
	case http.MethodDelete:
		return semconv.HTTPRequestMethodDelete
	case http.MethodGet:
		return semconv.HTTPRequestMethodGet
	case http.MethodHead:
		return semconv.HTTPRequestMethodHead
	case http.MethodOptions:
		return semconv.HTTPRequestMethodOptions
	case http.MethodPatch:
		return semconv.HTTPRequestMethodPatch
	case http.MethodPost:
		return semconv.HTTPRequestMethodPost
	case http.MethodPut:
		return semconv.HTTPRequestMethodPut
	case http.MethodTrace:
		return semconv.HTTPRequestMethodTrace
	default:
		return semconv.HTTPRequestMethodOther
	}
}
