package security

import (
	"context"
)

// TokenSigner signs and verifies cryptographically secure tokens.
type TokenSigner interface {
	Generate(ctx context.Context, subject string) (string, error)
	Verify(ctx context.Context, token string, payload []byte) (bool, error)
}
