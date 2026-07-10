package cleanup

import (
	"context"
	"time"
)

func RunLoop(stopCleanup <-chan struct{}, done chan<- struct{}, interval time.Duration, fn func(ctx context.Context)) {
	defer close(done)
	for {
		select {
		case <-stopCleanup:
			return
		case <-time.After(interval):
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			fn(ctx)
			cancel()
		}
	}
}
