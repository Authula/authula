package bootstrap

import (
	"sync"

	"github.com/Authula/authula/models"
)

// logLevel identifies one of the four severity methods exposed by
// models.Logger. It exists so the fan-out never has to depend on slog.
type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
)

// emit calls the matching method on the supplied logger.
func (l logLevel) emit(logger models.Logger, msg string, args ...any) {
	switch l {
	case levelDebug:
		logger.Debug(msg, args...)
	case levelInfo:
		logger.Info(msg, args...)
	case levelWarn:
		logger.Warn(msg, args...)
	case levelError:
		logger.Error(msg, args...)
	}
}

// fanoutLogger forwards every log call to a base logger plus any sinks attached
// at runtime. Sinks let plugins (for example the OTel log bridge) receive the
// same records without core knowing about them.
type fanoutLogger struct {
	mu    sync.RWMutex
	base  models.Logger
	sinks []*logSink
}

type logSink struct {
	logger models.Logger
}

func newFanoutLogger(base models.Logger) *fanoutLogger {
	return &fanoutLogger{base: base}
}

// AddLogger attaches sink and returns an idempotent detach function. A nil sink
// is ignored and detach becomes a no-op.
func (f *fanoutLogger) AddLogger(sink models.Logger) (detach func()) {
	if sink == nil {
		return func() {}
	}

	attached := &logSink{logger: sink}

	f.mu.Lock()
	f.sinks = append(f.sinks, attached)
	f.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			defer f.mu.Unlock()

			for i, candidate := range f.sinks {
				if candidate == attached {
					f.sinks = append(f.sinks[:i], f.sinks[i+1:]...)
					return
				}
			}
		})
	}
}

func (f *fanoutLogger) Debug(msg string, args ...any) {
	f.dispatch(levelDebug, msg, args...)
}

func (f *fanoutLogger) Info(msg string, args ...any) {
	f.dispatch(levelInfo, msg, args...)
}

func (f *fanoutLogger) Warn(msg string, args ...any) {
	f.dispatch(levelWarn, msg, args...)
}

func (f *fanoutLogger) Error(msg string, args ...any) {
	f.dispatch(levelError, msg, args...)
}

// dispatch snapshots the sinks under a read lock, then calls the base followed
// by every sink. Holding no lock while calling out keeps a slow or blocking
// logger from stalling attach/detach.
func (f *fanoutLogger) dispatch(level logLevel, msg string, args ...any) {
	sinks := f.snapshot()

	safeEmit(f.base, level, msg, args...)

	for _, sink := range sinks {
		if !safeEmit(sink.logger, level, msg, args...) {
			f.reportSinkFailure(level, msg)
		}
	}
}

func (f *fanoutLogger) snapshot() []*logSink {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sinks := make([]*logSink, len(f.sinks))
	copy(sinks, f.sinks)
	return sinks
}

// reportSinkFailure tells the base logger that a sink panicked. It is
// best-effort: the report itself is guarded so a broken base cannot panic the
// caller either.
func (f *fanoutLogger) reportSinkFailure(level logLevel, msg string) {
	if f.base == nil {
		return
	}
	safeEmit(f.base, levelError, "logger sink panicked", "level", level.String(), "message", msg)
}

// safeEmit calls logger and reports whether it returned normally. A panicking
// logger can never take down the request that was being logged.
func safeEmit(logger models.Logger, level logLevel, msg string, args ...any) (ok bool) {
	if logger == nil {
		return true
	}

	ok = true
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	level.emit(logger, msg, args...)
	return ok
}

// String returns the human-readable name of the level.
func (l logLevel) String() string {
	switch l {
	case levelDebug:
		return "debug"
	case levelInfo:
		return "info"
	case levelWarn:
		return "warn"
	case levelError:
		return "error"
	default:
		return "unknown"
	}
}
