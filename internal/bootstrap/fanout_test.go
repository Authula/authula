package bootstrap

import (
	"sync"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
)

func TestFanoutLoggerForwardsToBaseAndSinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
		emit  func(*fanoutLogger, string, ...any)
	}{
		{name: "debug", level: "debug", emit: func(f *fanoutLogger, msg string, args ...any) { f.Debug(msg, args...) }},
		{name: "info", level: "info", emit: func(f *fanoutLogger, msg string, args ...any) { f.Info(msg, args...) }},
		{name: "warn", level: "warn", emit: func(f *fanoutLogger, msg string, args ...any) { f.Warn(msg, args...) }},
		{name: "error", level: "error", emit: func(f *fanoutLogger, msg string, args ...any) { f.Error(msg, args...) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := new(internaltests.LevelRecordingLogger)
			sink := new(internaltests.LevelRecordingLogger)
			fanout := newFanoutLogger(base)
			fanout.AddLogger(sink)

			tc.emit(fanout, "hello", "key", "value")

			for name, logger := range map[string]*internaltests.LevelRecordingLogger{
				"base": base,
				"sink": sink,
			} {
				records := logger.Records()
				if len(records) != 1 {
					t.Fatalf("%s received %d records, want 1", name, len(records))
				}
				if records[0].Level != tc.level {
					t.Fatalf("%s level = %q, want %q", name, records[0].Level, tc.level)
				}
				if records[0].Message != "hello" {
					t.Fatalf("%s message = %q, want %q", name, records[0].Message, "hello")
				}
			}
		})
	}
}

func TestFanoutLoggerAddsSinkAfterConstruction(t *testing.T) {
	t.Parallel()

	base := new(internaltests.LevelRecordingLogger)
	sink := new(internaltests.LevelRecordingLogger)
	fanout := newFanoutLogger(base)

	fanout.Info("before")
	fanout.AddLogger(sink)
	fanout.Info("after")

	if got := sink.Messages(); len(got) != 1 || got[0] != "after" {
		t.Fatalf("sink messages = %v, want [after]", got)
	}
	if got := base.Messages(); len(got) != 2 {
		t.Fatalf("base received %d messages, want 2", len(got))
	}
}

func TestFanoutLoggerDetachStopsDeliveryAndIsIdempotent(t *testing.T) {
	t.Parallel()

	base := new(internaltests.LevelRecordingLogger)
	sink := new(internaltests.LevelRecordingLogger)
	fanout := newFanoutLogger(base)

	detach := fanout.AddLogger(sink)
	fanout.Info("first")
	detach()
	detach()
	fanout.Info("second")

	if got := sink.Messages(); len(got) != 1 || got[0] != "first" {
		t.Fatalf("sink messages = %v, want [first]", got)
	}
	if len(fanout.snapshot()) != 0 {
		t.Fatalf("sinks remaining = %d, want 0", len(fanout.snapshot()))
	}
}

func TestFanoutLoggerIgnoresNilSink(t *testing.T) {
	t.Parallel()

	base := new(internaltests.LevelRecordingLogger)
	fanout := newFanoutLogger(base)

	detach := fanout.AddLogger(nil)
	detach()
	detach()
	fanout.Info("hello")

	if len(fanout.snapshot()) != 0 {
		t.Fatalf("sinks remaining = %d, want 0", len(fanout.snapshot()))
	}
	if got := base.Messages(); len(got) != 1 {
		t.Fatalf("base received %d messages, want 1", len(got))
	}
}

func TestFanoutLoggerRecoversPanickingSink(t *testing.T) {
	t.Parallel()

	base := new(internaltests.LevelRecordingLogger)
	healthy := new(internaltests.LevelRecordingLogger)
	fanout := newFanoutLogger(base)

	fanout.AddLogger(new(internaltests.PanicLogger))
	fanout.AddLogger(healthy)

	fanout.Info("still delivered")

	if got := healthy.Messages(); len(got) != 1 || got[0] != "still delivered" {
		t.Fatalf("healthy sink messages = %v, want [still delivered]", got)
	}

	records := base.Records()
	if len(records) != 2 {
		t.Fatalf("base received %d records, want 2 (original + panic report)", len(records))
	}
	if records[1].Level != "error" || records[1].Message != "logger sink panicked" {
		t.Fatalf("panic report = %+v, want error record", records[1])
	}
}

func TestFanoutLoggerConcurrentAttachAndLog(t *testing.T) {
	t.Parallel()

	fanout := newFanoutLogger(new(internaltests.LevelRecordingLogger))

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 100 {
				fanout.Info("message")
			}
		}()
		go func() {
			defer wg.Done()
			sink := new(internaltests.LevelRecordingLogger)
			detach := fanout.AddLogger(sink)
			fanout.Info("message")
			detach()
		}()
	}
	wg.Wait()

	if got := len(fanout.snapshot()); got != 0 {
		t.Fatalf("sinks remaining = %d, want 0", got)
	}
}
