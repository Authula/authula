package util

import "github.com/GoBetterAuth/go-better-auth/models"

// PreserveNonSerializableFieldsOnConfig safely preserves all non-serializable fields from the source config
// into the target config.
func PreserveNonSerializableFieldsOnConfig(target, source *models.Config) {
	if source == nil || target == nil {
		return
	}

	// Preserve non-serializable fields from source config
	target.Mode = source.Mode
	target.DB = source.DB
	target.Logger.Logger = source.Logger.Logger
	target.SecondaryStorage.Storage = source.SecondaryStorage.Storage
	target.EndpointHooks = source.EndpointHooks
	target.DatabaseHooks = source.DatabaseHooks
	target.EventHooks = source.EventHooks
	target.Plugins = source.Plugins
	target.EventBus.PubSub = source.EventBus.PubSub
	// Preserve nested function fields
	target.EmailPassword.SendResetPasswordEmail = source.EmailPassword.SendResetPasswordEmail
	target.EmailPassword.Password.Hash = source.EmailPassword.Password.Hash
	target.EmailPassword.Password.Verify = source.EmailPassword.Password.Verify
	target.EmailVerification.SendVerificationEmail = source.EmailVerification.SendVerificationEmail
	target.User.ChangeEmail.SendEmailChangeVerificationEmail = source.User.ChangeEmail.SendEmailChangeVerificationEmail
}

// RequiresRestart checks if the configuration changes require a server restart.
// Returns true if critical fields have changed that affect routes or plugins.
func RequiresRestart(current, updated *models.Config) bool {
	if current == nil || updated == nil {
		return false
	}

	// Check if critical fields that affect routing or plugins have changed

	if current.BaseURL != updated.BaseURL {
		return true
	}

	if current.BasePath != updated.BasePath {
		return true
	}

	if current.EventBus.PubSubType != updated.EventBus.PubSubType {
		return true
	}

	return false
}
