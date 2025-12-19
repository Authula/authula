package signout

import (
	"context"
)

type SignOutUseCase interface {
	SignOut(ctx context.Context, sessionToken string) error
}
