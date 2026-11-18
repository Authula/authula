package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitForSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		signal func() <-chan struct{}
	}{
		{
			name: "returns when the signal is already closed",
			signal: func() <-chan struct{} {
				ch := make(chan struct{})
				close(ch)
				return ch
			},
		},
		{
			name: "returns when the signal closes later",
			signal: func() <-chan struct{} {
				ch := make(chan struct{})
				go close(ch)
				return ch
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signal := tt.signal()

			start := time.Now()
			WaitForSignal(t, signal)

			assert.Less(t, time.Since(start), backgroundTaskTimeout)
		})
	}
}
