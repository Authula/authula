# Dropbox Provider

Authula supports Dropbox as a built-in OAuth2 provider using [Dropbox OpenID Connect](https://docs.dropboxapi.com/dropbox-api/docs/oidc).

## Endpoints and scopes

| Item | Value |
|---|---|
| Authorization URL | `https://www.dropbox.com/oauth2/authorize` |
| Token URL | `https://api.dropboxapi.com/oauth2/token` |
| User info URL | `POST https://api.dropboxapi.com/2/openid/userinfo` |
| Default scopes | `openid`, `profile`, `email` |

The user info request sends an empty JSON object with `Authorization: Bearer <access_token>`.

## Create a Dropbox app

1. Open the [Dropbox App Console](https://www.dropbox.com/developers/apps) and create a scoped app.
2. Under **Permissions**, enable `openid`, `profile`, and `email`.
3. Under **OAuth 2 / Redirect URIs**, register the exact callback URL, including the API base path:
   `https://api.example.com/api/auth/oauth2/callback/dropbox`
4. Copy the **App key** and **App secret**.

## Configure credentials

Set the environment variables:

```text
DROPBOX_CLIENT_ID=<app key>
DROPBOX_CLIENT_SECRET=<app secret>
```

Or configure the provider in TOML:

```toml
[plugins.oauth2.providers.dropbox]
client_id = "..."
client_secret = "..."
redirect_url = "https://api.example.com/api/auth/oauth2/callback/dropbox"
scopes = ["openid", "profile", "email"]
```

When `redirect_url` is omitted, Authula builds it from `base_url` and `base_path` as `/oauth2/callback/dropbox`.

## Behavior

- `sub` (`dbid:...`) is the stable Dropbox account identifier. It is used for account linking because email addresses can change.
- `email` is normalized to lowercase.
- Dropbox OIDC guarantees `email_verified = true` for returned emails.
- The display name comes from `name`, falling back to `given_name` and `family_name`.
- PKCE defaults to enabled and can only be disabled per provider, which is not OAuth 2.1 compliant.
- Dropbox OIDC user info does not return an avatar, so `Picture` is empty.
- Refresh tokens and `token_access_type=offline` are not requested by default because this provider is intended for sign-in.
