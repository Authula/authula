package tests

import (
	"testing"
	"time"
)

const backgroundTaskTimeout = 5 * time.Second

// WaitForSignal blocks until signal is closed. It fails the test if the
// background task does not signal within backgroundTaskTimeout.
func WaitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(backgroundTaskTimeout):
		t.Fatal("timed out waiting for background task")
	}
}
