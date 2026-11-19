# Changelog

## Unreleased

### Changed

- JWT signing keys are owned by the JWT plugin. Keys are purpose-scoped (`jwt_plugin`, `oidc_id_token`, `oidc_access_token`) with an explicit active flag, persisted algorithm, and first-class `kid`. Other plugins consume the keys through the single public `services.JWTService` contract and the `models.SigningKey` model without importing the JWT plugin.
- The JWT plugin's `jwks` schema now stores the key id, algorithm, purpose, and active flag, and is created by the JWT plugin migration.
- The bearer plugin declares a dependency on the JWT plugin. Enabling it without `plugins.jwt.enabled = true` is rejected during plugin initialization.
- The JWT plugin now signs tokens with the shared default `RS256` instead of `EdDSA`. Token claims, lifetimes, refresh behaviour, and the public API are unchanged.
- JWKS responses publish each key's stored algorithm instead of a hard-coded `EdDSA` value.

### Fixed

- MySQL migration metadata writes no longer use a row alias, which MySQL rejects on plain `INSERT` statements.
- JWT token generation now uses the caller's context instead of a detached background context.
- Missing JWT signing keys preserve `models.ErrSigningKeyNotFound` so callers can match them with `errors.Is`.

### Upgrade note

The JWT `jwks` table schema changed in place. Local development databases and plugin migration history must be reset before running the new version.
