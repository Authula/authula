// Package tests holds shared test helpers for the OpenTelemetry plugin.
package tests

import (
	"context"
	"sync"

	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Exporter is an in-memory sdklog.Exporter that keeps exported records so
// tests can assert on them without a network endpoint.
type Exporter struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (e *Exporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range records {
		e.records = append(e.records, records[i].Clone())
	}
	return nil
}

func (e *Exporter) Shutdown(context.Context) error { return nil }

func (e *Exporter) ForceFlush(context.Context) error { return nil }

// Records returns a copy of every exported record, in order.
func (e *Exporter) Records() []sdklog.Record {
	e.mu.Lock()
	defer e.mu.Unlock()

	records := make([]sdklog.Record, len(e.records))
	copy(records, e.records)
	return records
}
