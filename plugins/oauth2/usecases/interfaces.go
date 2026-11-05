package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// AuthorizeUseCase is the handler-facing contract for starting an OAuth2 flow.
type AuthorizeUseCase interface {
	Authorize(ctx context.Context, req *types.AuthorizeRequest) (*AuthorizeResult, error)
}

// CallbackUseCase is the handler-facing contract for completing an OAuth2 flow.
type CallbackUseCase interface {
	Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error)
}
