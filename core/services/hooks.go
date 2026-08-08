package services

import (
	"github.com/Authula/authula/models"
)

// runHooks executes hooks in order and returns the first error encountered.
func runHooks[T any](hooks []models.ServiceHook[T], obj *T) error {
	for _, hook := range hooks {
		if err := hook(obj); err != nil {
			return err
		}
	}
	return nil
}

// runAfterHooks executes hooks in order without blocking the operation.
// Errors are logged and swallowed so that a failing hook does not report
// failure to the caller after the entity has already been persisted.
func runAfterHooks[T any](logger models.Logger, hookName string, hooks []models.ServiceHook[T], obj *T) {
	for _, hook := range hooks {
		if err := hook(obj); err != nil && logger != nil {
			logger.Error(hookName+" hook failed", "error", err.Error())
		}
	}
}
