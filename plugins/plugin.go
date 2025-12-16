package plugins

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// WrapEventHook wraps a typed event hook into a generic PluginEventHookFunc.
func WrapEventHook[T any](hook models.TypedPluginEventHook[T]) models.PluginEventHookFunc {
	return func(ctx *models.PluginContext, payload models.PluginEventHookPayload) error {
		typedPayload, ok := payload.(T)
		if !ok {
			return fmt.Errorf(
				"invalid event payload type: expected %T, got %T",
				*new(T),
				payload,
			)
		}
		return hook(ctx, typedPayload)
	}
}
