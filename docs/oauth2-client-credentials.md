# OAuth2 Client Credentials Tokens

Authula's OAuth2 plugin implements the OAuth 2.0 Client Credentials grant
([RFC 6749 section 4.4](https://www.rfc-editor.org/rfc/rfc6749#section-4.4)) for
confidential machine clients. The endpoint authenticates the client and returns
a signed JWT access token. It never returns a refresh token or an ID token.

Access tokens follow the OAuth 2.0 JWT Access Token profile
([RFC 9068](https://www.rfc-editor.org/rfc/rfc9068)): every access token carries
`typ: at+jwt` and the required `client_id`, `jti`, `iat`, `sub`, `aud`, `exp`,
and `iss` claims.

## Prerequisites

- `authorization_server_enabled = true` in `[plugins.oauth2]`.
- The `jwt` plugin must be enabled. It owns the signing keys used by the token
  endpoint. With the authorization server enabled, the OAuth2 plugin declares a
  dependency on `jwt`, so `authula migrate --plugins oauth2` also requires
  `jwt` to be selected.
- `allowed_scopes` must contain at least one non-OIDC scope. The default
  `["openid", "profile", "email"]` ceiling is rejected at startup when the
  authorization server is enabled because client credentials cannot request any
  of those scopes.
- Clients must be provisioned ahead of time. There is no client management API
  in this release.

## Endpoint

```text
POST {base_url}{base_path}/oauth2/token
Content-Type: application/x-www-form-urlencoded
```

The endpoint is public by design. There is no route middleware: clients
authenticate with their client credentials, not a user session.

## Client authentication

Both `client_secret_basic` ([RFC 6749 section 2.3.1](https://www.rfc-editor.org/rfc/rfc6749#section-2.3.1))
and `client_secret_post` are supported globally.

### HTTP Basic

```bash
curl -X POST https://api.example.com/api/auth/oauth2/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -u 'my-client-id:my-client-secret' \
  -d 'grant_type=client_credentials&scope=orders:read orders:write'
```

Construct the Basic header exactly as RFC 6749 section 2.3.1 requires:

1. URL-form-encode the client ID by itself.
2. URL-form-encode the client secret by itself.
3. Join them as `encoded_id:encoded_secret`.
4. Base64-encode the result.

The server base64-decodes the header, splits on the first `:`, and
URL-form-decodes both values. This preserves spaces, `%`, `&`, `+`, and `:`
inside the credentials.

### POST body

```bash
curl -X POST https://api.example.com/api/auth/oauth2/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=client_credentials' \
  -d 'client_id=my-client-id' \
  -d 'client_secret=my-client-secret' \
  -d 'scope=orders:read orders:write'
```

Do not mix Basic authentication with `client_id` or `client_secret` form
parameters. Combining them returns `invalid_request`.

Credentials in the query string are ignored. Only the form body is read.

## Scope semantics

`scope` is optional and there are no default scopes.

| Request | Result |
|---|---|
| `scope` omitted | A token with an empty scope set. It authorizes nothing. |
| `scope=read write` | A token granting exactly `read` and `write`. |
| `scope=` (present but empty) | `400 invalid_scope`. |
| Duplicate, repeated, or leading/trailing spaces | `400 invalid_scope`. |
| `*` or `prefix*` | `400 invalid_scope`. |
| Any OIDC scope | `400 invalid_scope`. |

Requested scopes must be an exact, case-sensitive subset of both:

1. the client's `allowed_scopes`, and
2. the deployment's `plugins.oauth2.allowed_scopes`.

Wildcards are never valid for machine tokens. The access-control plugin still
supports `*` and `prefix*` for user API keys, but machine tokens must list the
exact permission keys.

All OpenID Connect scopes are rejected for this grant:

- `openid`
- `profile`
- `email`
- `address`
- `phone`
- `offline_access`

The deployment ceiling may include OIDC scopes for future authorization-code
flows. They are rejected at token issuance for `client_credentials`.

## Success response

```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6Ii4uLiJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "orders:read orders:write"
}
```

`scope` is always a space-delimited string, never an array. It is omitted when
the granted scope set is empty, even if the request omitted `scope`.

`Cache-Control: no-store` and `Pragma: no-cache` are set on every response,
including errors.

## Error responses

OAuth error bodies use the standard shape:

```json
{
  "error": "invalid_scope",
  "error_description": "requested scope is not allowed"
}
```

| Condition | Status | Error |
|---|---:|---|
| Missing, invalid, or multiple `Content-Type` | 400 | `invalid_request` |
| Duplicate parameters | 400 | `invalid_request` |
| Basic and body credentials together | 400 | `invalid_request` |
| Missing `grant_type` | 400 | `invalid_request` |
| Unsupported grant type | 400 | `unsupported_grant_type` |
| Missing body credentials | 400 | `invalid_client` |
| Unknown, inactive, or soft-deleted client, or wrong secret, with body credentials | 400 | `invalid_client` |
| Malformed Basic credentials or multiple `Authorization` headers | 401 | `invalid_client` |
| Unknown, inactive, or soft-deleted client, or wrong secret, with Basic credentials | 401 | `invalid_client` |
| Client lacks `client_credentials` in `allowed_grants` | 400 | `unauthorized_client` |
| Invalid, empty, duplicate, wildcard, OIDC, or escalated scope | 400 | `invalid_scope` |
| Internal token issuance failure | 500 | `server_error` |

An internal failure still uses the OAuth error body:

```json
{
  "error": "server_error",
  "error_description": "The authorization server encountered an unexpected condition."
}
```

Every 401 includes:

```http
WWW-Authenticate: Basic realm="oauth2/token", charset="UTF-8"
```

## Token claims

Access tokens use the RFC 9068 profile. The protected header is:

```json
{"alg":"EdDSA","kid":"...","typ":"at+jwt"}
```

Machine access tokens are signed with the JWT plugin's active key and carry:

| Claim | Value |
|---|---|
| `iss` | Configured JWT issuer. |
| `aud` | Configured JWT audience. |
| `sub` | OAuth client ID. |
| `client_id` | OAuth client ID. |
| `iat`, `exp`, `jti` | Issued-at, expiry, and unique token ID. |
| `token_type` | `access_token`. |
| `actor_type` | `machine`. |
| `scope` | Canonical RFC 9068 space-delimited scope string. Omitted when empty. |
| `organization_id` | Present when the client belongs to an organization. |

Never present: `user_id`, `session_id`, refresh-token claims, or ID-token
claims.

The effective lifetime is `plugins.oauth2.client_credentials_token_ttl`
(default `15m`) and is reflected in both `exp - iat` and the response
`expires_in`.

### First-party user tokens

User access tokens issued by Authula use the same RFC 9068 profile and header.
They add `client_id` with the reserved first-party identifier:

```toml
[plugins.jwt]
access_token_client_id = "authula"
```

The default is `authula`. Machine tokens always use the real OAuth client ID.
Future authorization-code tokens must also use the real OAuth client ID.

Refresh tokens do not use `typ: at+jwt` and do not carry `client_id`. Future
OpenID Connect ID tokens must use `typ: JWT`. Bearer access-token validation
rejects `typ: JWT`, so an ID token cannot be used as an access token.

### Validation requirements

`ValidateToken` accepts only tokens that meet the profile:

- protected header `typ` is `at+jwt` or `application/at+jwt`;
- protected header `alg` is `EdDSA` (`alg: none` is rejected);
- `iss`, `aud`, and `exp` are valid for this deployment;
- `sub`, `client_id`, `iat`, and `jti` are present;
- `token_type` is `access_token`;
- `actor_type` is `user` or `machine`;
- `scope`, when present, is a valid space-delimited OAuth scope string.

There is no separate `scopes` array claim. The bearer and access-control layers
parse `scope` into a slice when they perform permission checks. This keeps
`scope` as the single source of truth.

`ValidateRefreshToken` does not require `typ: at+jwt`.

### RS256 conformance gap

Strict RFC 9068 conformance requires RS256 support. Authula currently signs
with EdDSA only. RS256 key generation, JWKS publication, rotation, algorithm
selection, and validation are a documented follow-up. Deployments that need
strict RFC 9068 interoperability should wait for that work.

### Compatibility

Previously issued development access tokens do not carry `typ: at+jwt`. They
are rejected after this change. There is no backwards-compatibility mode.

## Resource server validation

Use the bearer plugin to validate machine tokens and the access-control plugin
to enforce scopes. Machine-token scopes must exactly match access-control
permission keys. Machine tokens never receive wildcard scopes and wildcards are
not accepted when enforcing machine-token permissions.

```toml
[[route_mappings]]
paths = ["GET:/api/data"]
plugins = ["bearer.auth"]
permissions = ["read"]
```

A client requesting `scope=read` can call `GET:/api/data`. A client that omits
`scope` receives an empty scope set and is denied.

## Provisioning a client

Clients are created from deployment code with the OAuth2 repositories and the
SHA-256 client secret service. Never store or seed plaintext secrets.

```go
secrets := services.NewSHA256ClientSecretService("authula_cs_")

generated, err := secrets.Generate()
if err != nil {
    return err
}

repo := repositories.NewBunOAuthClientRepository(db)
_, err = repo.Create(ctx, &types.OAuthClient{
    ID:                 util.GenerateUUID(),
    ClientID:           "my-client-id",
    ClientSecretHash:   generated.Hash,
    ClientSecretSalt:   generated.Salt,
    ClientSecretPrefix: generated.Prefix,
    ClientSecretStart:  generated.Start,
    ClientSecretLast:   generated.Last,
    Name:               "My machine client",
    RedirectURIs:       []string{},
    AllowedGrants:      []string{types.OAuthGrantTypeClientCredentials},
    AllowedScopes:      []string{"orders:read", "orders:write"},
    OwnerType:          types.OAuthClientOwnerTypeOrganization,
    OwnerID:            "org-1",
    CreatorID:          "user-1",
    IsActive:           true,
})
if err != nil {
    return err
}
// Return generated.Value once to the operator. It cannot be recovered later.
```

Client secrets use 48 random URL-safe characters (about 288 bits) prefixed with
the configured value and are stored as salted SHA-256 hashes. Never log the raw
secret. Rotation is a follow-up.

### Stored display metadata

- `plugins.oauth2.client_secret_value_prefix` is the literal prefix used when
  generating a secret. Default: `authula_cs_`.
- `oauth_clients.client_secret_prefix` stores the configured prefix captured at
  creation time, not a derived hint.
- `oauth_clients.client_secret_start` is the first 4 characters of the random
  secret portion.
- `oauth_clients.client_secret_last` is the last 4 characters of the random
  secret portion.

These fields let an operator identify a secret without storing it.

## Operational notes

- **Schema reset.** The initial OAuth2 migration now creates
  `client_secret_start` and `client_secret_last`. The schema is not in
  production. Development databases that already applied the OAuth2 migration
  must drop and recreate the OAuth tables so the new `NOT NULL` columns exist.
- **Access-token compatibility.** Tokens issued before the RFC 9068 change do
  not have `typ: at+jwt` and are rejected. This is intentional.
- **No revocation or introspection.** JWTs remain valid until `exp`. Disabling
  or deleting a client stops new tokens only. Existing tokens keep their
  granted scopes until expiry.
- **No default scopes.** A request without `scope` succeeds with an empty scope
  set. Clients should send `scope` explicitly.
- **Scope ceiling.** `allowed_scopes` defaults to `openid profile email`.
  Deployments enabling client credentials must add their API scopes.
- **Rate limiting.** Add a custom rate-limit rule for
  `POST:/api/auth/oauth2/token`. The key is the full path including the base
  path, and the limit is applied per IP.
- **JWT dependency.** `authorization_server_enabled = true` requires the JWT
  plugin. Startup fails if the token issuer is unavailable.
- **Follow-ups.** RS256 signing and strict RFC 9068 interoperability, client
  management, secret rotation, revocation, and introspection are separate
  pieces of work.
