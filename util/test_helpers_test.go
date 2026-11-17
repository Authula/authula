package util

import "sync"

type testLogRecord struct {
	Level   string
	Message string
	Args    []any
}

// recordingLogger captures log calls. It is safe for concurrent use because
// async utilities may log from a background goroutine.
type recordingLogger struct {
	mu      sync.Mutex
	records []testLogRecord
}

func (l *recordingLogger) Debug(msg string, args ...any) { l.record("debug", msg, args...) }
func (l *recordingLogger) Info(msg string, args ...any)  { l.record("info", msg, args...) }
func (l *recordingLogger) Warn(msg string, args ...any)  { l.record("warn", msg, args...) }
func (l *recordingLogger) Error(msg string, args ...any) { l.record("error", msg, args...) }

func (l *recordingLogger) record(level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, testLogRecord{Level: level, Message: msg, Args: args})
}

func (l *recordingLogger) Records() []testLogRecord {
	l.mu.Lock()
	defer l.mu.Unlock()

	records := make([]testLogRecord, len(l.records))
	copy(records, l.records)
	return records
}
