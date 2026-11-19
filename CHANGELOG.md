# Changelog

## Unreleased

### Changed

- JWT signing keys are owned by the JWT plugin. Keys are purpose-scoped (`jwt_plugin`, `oidc_id_token`, `oidc_access_token`) with an explicit active flag, persisted algorithm, and first-class `kid`. Other plugins consume the keys through the single public `services.JWTService` contract and the `models.SigningKey` model without importing the JWT plugin.
- The JWT plugin's `jwks` schema now stores the key id, algorithm, purpose, and active flag, and is created by the JWT plugin migration.
- The bearer and OAuth2 plugins declare a dependency on the JWT plugin. Enabling either without `plugins.jwt.enabled = true` is rejected during plugin initialization.
- The JWT plugin now signs tokens with the shared default `RS256` instead of `EdDSA`. Token claims, lifetimes, refresh behaviour, and the public API are unchanged.
- JWKS responses publish each key's stored algorithm instead of a hard-coded `EdDSA` value.
- The OAuth2 authorization-server schema was refactored in place to add client policy fields, authorization requests, one-time consent challenges, access grants, and rotating refresh tokens. The unused `authorization_codes` and `oauth_login_transactions` tables were removed.
- OIDC authentication context lives in OAuth2-owned authorization-request and token records, not in core sessions. Core sessions are unchanged.

### Fixed

- MySQL migration metadata writes no longer use a row alias, which MySQL rejects on plain `INSERT` statements.

### Upgrade note

The JWT `jwks` table and the OAuth2 authorization-server schema changed in place, with no backward-compatibility migrations. Reset local development databases and plugin migration history before running the new version.
