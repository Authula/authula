# OpenTelemetry plugin

Opt-in distributed tracing and metrics for Authula, backed by the OpenTelemetry
SDK. The plugin extracts and propagates W3C trace context and records request
and runtime metrics.

Telemetry is **fail-open**: the hooks never return an error, so a telemetry
problem can never short-circuit a real request. Invalid configuration or a
provider/exporter failure, however, fails `Init` so the server refuses to start
rather than running blind.

## Enabling

A present `[plugins.otel]` section enables the plugin. Set `enabled = false` to
turn it back off. When disabled the plugin initializes nothing: no providers, no
exporters, no goroutines, and no hooks.

```toml
[plugins.otel]
enabled = true
service_name = "authula-service"
exporter = "otlp"
endpoint = "http://localhost:4318"
insecure = true
```

## Configuration reference

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | `true` when the section exists | Master switch. |
| `service_name` | string | `authula` | `service.name` resource attribute. |
| `exporter` | string | `otlp` | `otlp`, `stdout`, or `none`. |
| `protocol` | string | `http` | `http`, `grpc`, or empty (env/default). |
| `endpoint` | string | empty | OTLP endpoint. Empty lets the exporter read `OTEL_EXPORTER_OTLP_ENDPOINT`. |
| `insecure` | bool | `false` | Disable transport security. |
| `headers` | map | empty | Extra OTLP headers. Empty lets `OTEL_EXPORTER_OTLP_HEADERS` apply. |
| `traces_enabled` | bool | `true` | Enable tracing. |
| `metrics_enabled` | bool | `true` | Enable request/runtime metrics. |
| `runtime_metrics` | bool | `true` | Enable Go runtime metrics. |
| `sample_ratio` | float | unset | Trace sample ratio in `[0,1]`. Omit to let the SDK read `OTEL_TRACES_SAMPLER`. |
| `trust_proxy` | bool | `false` | Trust `X-Forwarded-Proto` / `X-Forwarded-For`. |

### Precedence

One rule covers every setting: **config field > standard OpenTelemetry
environment variable > SDK default**.

- `service_name`: config → `OTEL_SERVICE_NAME` → `authula`.
- `endpoint` empty: the exporter reads `OTEL_EXPORTER_OTLP_ENDPOINT`.
- `protocol` empty: `OTEL_EXPORTER_OTLP_PROTOCOL`, otherwise `http`.
- `headers` empty: `OTEL_EXPORTER_OTLP_HEADERS` applies.
- `sample_ratio` unset: `OTEL_TRACES_SAMPLER` applies. When set, the plugin uses
  `ParentBased(TraceIDRatioBased(ratio))`; an explicit `0.0` samples nothing.

Exporter tuning (batch timeout/size, export interval, export timeouts) is left
to the SDK: `OTEL_BSP_*`, `OTEL_METRIC_EXPORT_INTERVAL`, and
`OTEL_EXPORTER_OTLP_*_TIMEOUT`.

## What is emitted

### Spans

- Name: `{method} {route}` for matched routes (e.g. `GET /api/auth/me`), or just
  `{method}` when no route matched. The raw URL path is never used as the span
  target.
- Kind: server.
- The `traceparent` header is extracted and propagated, so spans join an
  existing trace. The span is placed on the request context, so handlers can use
  `trace.SpanFromContext(r.Context())` and create child spans.

Span attributes: `http.request.method`, `http.route` (method-stripped; fixed
`unmatched` when no route matched), `url.path`, `url.scheme`, `client.address`,
`http.response.status_code`, and `authula.actor.type` / `authula.actor.id` when
an actor is present.

Status semantics follow the stable HTTP semantic conventions: only `>= 500`
responses set the span status to `Error` and add `error.type` (the status code
as a string). A 4xx server span is not an error, so its status stays unset.

### Metrics

The only instrument is the semantic-convention histogram
`http.server.request.duration` (unit `s`, with the recommended bucket
boundaries). HTTP semantic conventions define no request-count metric, so none
is emitted; read request counts from the histogram's count aggregation.

Recorded attributes: `http.request.method`, `http.route`,
`http.response.status_code`, `url.scheme`, plus `error.type` for `>= 500`
responses. OPTIONS preflight requests are not recorded.

`runtime_metrics = true` additionally starts the OpenTelemetry runtime
instrumentation (Go runtime metrics) on the same meter provider.

## Bypass paths

Requests that bypass the normal lifecycle are traced too: OPTIONS preflight,
`isRequestDisabled` 404s, and `DisabledPaths` 404s. The on-request hook opts into
bypass paths with `RunOnBypass: true`; the on-complete hook always runs and sees
the final status.

## Why hooks instead of `otelhttp`

Authula's routing and bypass rules live in the hook system, not in a plain
`http.Handler` chain. Using hooks keeps tracing consistent with every other
cross-cutting concern: it runs at a defined stage, respects the final status,
and covers short-circuits and bypass paths that an `otelhttp` middleware would
miss or double-count.

## Instance-local providers

The plugin holds its own `Tracer`, `Meter`, and propagator. It never calls
`otel.SetTracerProvider` / `otel.SetMeterProvider` / `otel.SetTextMapPropagator`.
This keeps tests isolated and avoids process-global state.

The trade-off: third-party code (or future plugins) that uses the *global*
`otel.Tracer(...)` is not captured. Only spans on `req.Context()` are. There is
intentionally no global-registration option.

## Dependencies

Adding the plugin links the OpenTelemetry SDK, exporters, and runtime
instrumentation into the binary even when the plugin is disabled (the factory
imports it unconditionally). It is "lightweight when disabled" at runtime, not
in binary size.
