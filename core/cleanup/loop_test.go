package cleanup_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/cleanup"
)

type cleanupCall struct {
	ctx         context.Context
	hasDeadline bool
	remaining   time.Duration
}

type contextRecorder struct {
	mu    sync.Mutex
	calls []cleanupCall
}

func (r *contextRecorder) record(ctx context.Context) {
	deadline, ok := ctx.Deadline()
	call := cleanupCall{ctx: ctx, hasDeadline: ok}
	if ok {
		call.remaining = time.Until(deadline)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, call)
}

func (r *contextRecorder) all() []cleanupCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	calls := make([]cleanupCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

func assertClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case _, ok := <-done:
		assert.False(t, ok, "done channel should be closed")
	default:
		t.Fatal("done channel is not closed")
	}
}

func assertNotClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case _, ok := <-done:
		if !ok {
			t.Fatal("done channel should not be closed")
		}
	default:
	}
}

func TestRunLoop_RunsAtEachInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		recorder := &contextRecorder{}
		stop := make(chan struct{})
		done := make(chan struct{})

		go cleanup.RunLoop(stop, done, 10*time.Millisecond, func(ctx context.Context) {
			calls.Add(1)
			recorder.record(ctx)
		})

		synctest.Wait()
		assert.Zero(t, calls.Load(), "cleanup must not run before the first interval")

		time.Sleep(25 * time.Millisecond)
		synctest.Wait()
		assert.Equal(t, int32(2), calls.Load())

		close(stop)
		<-done
		assertClosed(t, done)
		assert.Equal(t, int32(2), calls.Load())
	})
}

func TestRunLoop_StopBeforeFirstInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		stop := make(chan struct{})
		done := make(chan struct{})

		go cleanup.RunLoop(stop, done, time.Hour, func(context.Context) {
			calls.Add(1)
		})

		synctest.Wait()
		close(stop)
		<-done

		assert.Zero(t, calls.Load())
		assertClosed(t, done)
	})
}

func TestRunLoop_ProvidesCancelledTimeoutContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recorder := &contextRecorder{}
		stop := make(chan struct{})
		done := make(chan struct{})

		go cleanup.RunLoop(stop, done, 10*time.Millisecond, recorder.record)

		time.Sleep(15 * time.Millisecond)
		synctest.Wait()
		close(stop)
		<-done

		calls := recorder.all()
		require.Len(t, calls, 1)

		assert.True(t, calls[0].hasDeadline, "cleanup context should carry a deadline")
		assert.Equal(t, 30*time.Second, calls[0].remaining)
		assert.ErrorIs(t, calls[0].ctx.Err(), context.Canceled)
	})
}

func TestRunLoop_StopWaitsForInFlightCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{}, 1)
		release := make(chan struct{})
		var calls atomic.Int32
		stop := make(chan struct{})
		done := make(chan struct{})

		go cleanup.RunLoop(stop, done, 10*time.Millisecond, func(context.Context) {
			calls.Add(1)
			started <- struct{}{}
			<-release
		})

		time.Sleep(10 * time.Millisecond)
		<-started
		synctest.Wait()

		close(stop)
		synctest.Wait()
		assertNotClosed(t, done)

		close(release)
		<-done

		assertClosed(t, done)
		assert.Equal(t, int32(1), calls.Load())
	})
}

func TestRunLoop_NoCleanupAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		stop := make(chan struct{})
		done := make(chan struct{})

		go cleanup.RunLoop(stop, done, 10*time.Millisecond, func(context.Context) {
			calls.Add(1)
		})

		time.Sleep(25 * time.Millisecond)
		synctest.Wait()
		close(stop)
		<-done
		callsAtStop := calls.Load()

		time.Sleep(time.Hour)
		synctest.Wait()

		assert.Equal(t, callsAtStop, calls.Load())
	})
}
