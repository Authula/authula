# Sign-out

`POST /sign-out` revokes sessions for the authenticated user. Which sessions it revokes depends on the request body and on how the request was authenticated.

## Modes

| Request                                | What is revoked                                                        | Caller's cookie cleared |
| -------------------------------------- | ---------------------------------------------------------------------- | ----------------------- |
| No body                                | The session that authenticated this request (cookie, or the JWT's `session_id` claim) | Yes          |
| `{"session_id": "<id>"}`               | That one session, if it belongs to the caller                          | Only if it is the caller's own current session |
| `{"sign_out_all": true}`               | Every session for the user                                             | Yes                     |

`sign_out_all` takes precedence over `session_id` when both are supplied.

## Responses

| Status | When                                                          |
| ------ | ------------------------------------------------------------- |
| 200    | Sessions revoked                                              |
| 400    | No body and the request carries no session id (e.g. a JWT without a `session_id` claim) |
| 403    | `session_id` exists but belongs to a different user           |
| 404    | `session_id` does not exist                                   |
| 422    | `session_id` was supplied but empty                           |
| 500    | Unexpected storage error                                      |

Ownership mismatches return `403` rather than `404`. The caller is already authenticated and session ids are unguessable UUIDs, so hiding existence gains little, and a `403` is a clearer signal to the client.

## Cookie clearing

The session and CSRF plugins clear their cookies in an after-hook when the `auth.sign_out` request-context flag is set. The handler sets that flag only when the caller's **own** session was revoked (no-body path, `sign_out_all`, or `session_id` equal to the current session). Revoking a different session by id leaves the caller signed in.

## Go API

```go
result, err := coreAPI.SignOut(ctx, userID, currentSessionID, requestedSessionID, signOutAll)
```

- `currentSessionID` is the session that authenticated the request (`reqCtx.Values["session_id"]`, set by the session plugin), or `nil`.
- `requestedSessionID` and `signOutAll` come from the request body.
- `result.ClearedCurrentSession` reports whether the caller's own session was revoked.
