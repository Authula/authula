# Pagination

Authula's list endpoints return a page of results plus the metadata needed to walk
the rest. This page describes the contract, the bounds applied to caller input, and
when to reach for an unpaginated call instead.

The organizations plugin implements this contract on all five of its list endpoints.
Other plugins predate it: `plugins/api-key` uses its own flat envelope and
`plugins/admin` uses cursor pagination.

## Request

Two optional query parameters:

| Parameter | Default | Meaning |
| --- | --- | --- |
| `page` | `1` | 1-indexed page number |
| `limit` | `10` | Rows per page |

```
GET /organizations?page=2&limit=25
```

## Response

Every paginated endpoint returns the same envelope: a `data` array and a
`pagination` object.

```json
{
  "data": [ ... ],
  "pagination": {
    "page": 2,
    "limit": 25,
    "total": 138,
    "total_pages": 6,
    "has_more": true
  }
}
```

`total` counts every row matching the query, not just the current page. Iterate
until `has_more` is `false`.

## Bounds

Out-of-range input is **clamped, never rejected**. A list endpoint does not return
`400` for a bad `page` or `limit`; it coerces the value into range and serves the
request. The `pagination` object echoes the values actually applied, so a client can
always see what it got.

| Input | Result |
| --- | --- |
| `page` below 1 | `1` |
| `limit` below 1 | `10` (the default) |
| `limit` above the maximum | the maximum |
| unparseable value | the default for that parameter |

The ceiling exists to stop a single request from turning a paginated query into a
full table scan — a `limit` of a million would otherwise make the database
materialise the whole table, hold a pooled connection open for the duration, and
force the API process to serialise the result.

### Configuring the maximum

The ceiling defaults to **100** and is configurable per deployment:

```toml
[plugins.organizations]
max_page_limit = 250
```

A value of zero or less is treated as unset and falls back to 100. The floor is
applied before the ceiling, so a `max_page_limit` below 10 also caps the default
page size: with `max_page_limit = 5`, a request with no `limit` returns 5 rows.

## Ordering

Rows come back newest first, ordered by `created_at DESC, id DESC`. The `id`
tiebreaker keeps paging stable when several rows share a timestamp — without it, a
row can be skipped or repeated across page boundaries.

Pending-invitation lookups are the deliberate exception: they order oldest first,
because acceptance resolves role conflicts in favour of the earliest invitation.

## Fetching a whole collection

When embedding Authula as a Go library, every paginated `ListAll…` method has an
unpaginated `GetAll…` twin that takes no pagination and returns a plain slice:

```go
page, err := api.ListAllMembers(ctx, actor, orgID, pagination.Params{Page: 1, Limit: 50})
all, err := api.GetAllMembers(ctx, actor, orgID)
```

The `GetAll…` methods are **not** subject to `max_page_limit` — they are the
sanctioned way to ask for a whole collection, and they skip the `SELECT COUNT(*)`
that the paginated path needs, making them a single round-trip.

They are Go-only and are not exposed over HTTP. An HTTP client that needs more than
`max_page_limit` rows either pages through the results or runs against a deployment
configured with a higher ceiling.
