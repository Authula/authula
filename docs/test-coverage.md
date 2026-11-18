# Test Coverage

Status: **90% production coverage target met.**

Snapshot date: 2026-11-18.

## Baseline and result

| Metric | Statements | Coverage |
| --- | ---: | ---: |
| Start of coverage effort (`main` at f291938) | 11,637 / 13,779 | 84.5% |
| `main` at 7b00478 (#189–#197 merged) | 12,010 / 13,779 | 87.2% |
| All coverage PRs #198–#216 merged (`origin/main`, measured 2026-11-18) | **12,824 / 13,779** | **93.1%** |
| Target | 12,402 | 90.0% |

All 19 coverage PRs are merged into `origin/main`. The profile above comes from a full
`make test-coverage` run on the rebased branch.

Statements added by this effort: **+1,187** (84.5% → 93.1%).

## How to measure

```bash
make test-coverage-filtered
```

This runs the suite, writes `coverage.out` and `coverage.html`, and prints the filtered report.
For a standalone profile or a different target:

```bash
./scripts/coverage-filtered.sh coverage.out
COVERAGE_TARGET=95 ./scripts/coverage-filtered.sh coverage.out
```

## Scope

`scripts/coverage-filtered.sh` defines production scope. Excluded on purpose:

- `cmd/` entrypoints.
- `openapiexport/` export binary.
- `internal/tests/` and any `**/tests/` package (shared test support).
- Generated `openapi_docs.go` files, including all of `core/openapi`.

Do not add these back to inflate the number.

## Completed slices

28 test-only PRs. Every PR touches only `*_test.go` files (or new files under a package's
`tests/` folder) and adds no dependencies.

| PR | Slice | Target | Result |
| ---: | --- | --- | --- |
| #189 | `root-auth-api` | `auth.go` | 97.8% |
| #190 | `root-bootstrap` | `bootstrap.go` | 100% |
| #191 | `core-repo-account` | account repository | 98.3% |
| #192 | `core-repo-user` | user repository | 100% |
| #193 | `root-router` | `router.go` | 94.5% |
| #194 | `core-types` | `core/types` | 100% |
| #195 | `core-service-account` | account service | 100% |
| #196 | `internal-router` | deferred response writer | 100% |
| #197 | `core-service-password` | Argon2 + user service | 94.7% / 100% |
| #198 | `jwt-plugin` | JWT lifecycle, routes, migrations | 100% (package 93.9%) |
| #199 | `csrf-plugin` | CSRF lifecycle and headers | 98.0% |
| #200 | `jwt-types` | JWT types | 100% |
| #201 | `jwt-services` | token service + JWKS cache | 90.8% |
| #202 | `jwt-repositories` | refresh-token + JWKS repos | 98.7% |
| #203 | `jwt-handlers` | refresh handler | 92.3% |
| #204 | `session-plugin` | session lifecycle and hooks | 100% |
| #205 | `bearer-plugin` | bearer middlewares and hooks | 100% |
| #206 | `totp-plugin` | TOTP lifecycle, hooks, migrations | 96.0% |
| #207 | `email-plugin` | email lifecycle and delivery | 98.6% |
| #208 | `magic-link-plugin` | magic-link root package | 100% |
| #209 | `magic-link-usecases` | magic-link usecases | 100% |
| #210 | `migrations-migrator` | `migrator.go` | 100% |
| #211 | `core-events-bus` | `event_bus.go` | 100% |
| #212 | `migrations-registry` | core migrations + variants | 100% |
| #213 | `otel-exporters` | exporter/logger builders | 100% |
| #214 | `core-events-providers` | Watermill provider validation | 73.1% |
| #215 | `bootstrap-database` | DB init and pool config | 93.8% |
| #216 | `otel-plugin` | OTel plugin lifecycle and hooks | 98.8% (package) |

Results are the figures each PR reported when it was opened. Later PRs can raise a package's
number, so the table below is the current view.

## Remaining gaps

The overall target is already met. These packages are still below 90% and are the next
candidates for unit tests. Measured on `origin/main` with all coverage PRs (#198–#216) merged.
Deficit is the number of statements needed for that package to reach 90%.

| Package | Coverage | Statements | Deficit |
| --- | ---: | ---: | ---: |
| `plugins/email-password/usecases` | 74.2% | 279/376 | +60 |
| `plugins/organizations/services` | 86.7% | 880/1015 | +34 |
| `plugins/email-password` | 65.5% | 55/84 | +21 |
| `plugins/email-password/handlers` | 79.2% | 152/192 | +21 |
| `core/events` | 86.0% | 265/308 | +13 |
| `openapi` | 79.0% | 79/100 | +11 |
| `plugins/email-password/services` | 77.0% | 47/61 | +8 |
| `core/email/template` | 82.7% | 43/52 | +4 |
| `plugins/otel/types` | 87.5% | 21/24 | +1 |

Total deficit: **173 statements** across 9 packages.

### Excluded: OAuth2

All `plugins/oauth2/...` packages are left out on purpose. The plugin is being rewritten as a full
OAuth 2.1 / OIDC plugin that implements the RFCs listed in the OAuth2 plugin docs. Tests written
against the current code would be thrown away, so measure and test OAuth2 after that rewrite lands.

### Uncovered code by package

Start with the largest deficits. Functions listed are at 0% or below 90%, from the coverage profile.

**`plugins/email-password/usecases`** (+60)

- 0%: `GetSessionByID` and `GetUserByID` (`sign_in_usecase.go`).
- Under 75%: `ChangePassword` (70.0%), `sendChangedPasswordEmail` (60.0%), `RequestChange` (71.4%),
  `sendRequestEmailChangeEmail` (60.0%), `sendEmailVerification` (60.0%), `SignIn` (74.1%),
  `VerifyEmail` (72.4%), `handleEmailVerification` (71.4%), `handleEmailChangeVerificationEmail`
  (68.8%), `Send` (75.0%), `sendChangedEmailEmails` (75.0%).
- Under 80%: `SignUp` (78.0%), `RequestReset` (78.6%).

**`plugins/organizations/services`** (+34)

- Under 70%: `authorizeRemoveMemberWeight` (53.3%), `GetOrganizationInvitation` (60.0%),
  `RevokeOrganizationInvitation` (62.5%), `DeleteOrganization` (66.7%), `authorizeRoleWeight`
  (66.7%), `UpdateOrganization` (67.6%), `GetTeam` (70.0%), `RemoveTeamMember` (71.4%).
- 70–80%: `CreateTeam` (72.2%), `DeleteTeam` (72.2%), `NewOrganizationTeamService` (75.0%),
  `NewOrganizationMemberService` (75.0%), `CreateOrganization` (80.8%), `RemoveMember` (81.0%),
  `AcceptOrganizationInvitation` (79.3%), `AcceptPendingOrganizationInvitationsForEmail` (78.6%).

**`plugins/email-password`** root package (+21)

- 0%: the `api.go` wrappers (`SignUp`, `SignIn`, `VerifyEmail`, `SendEmailVerification`,
  `RequestPasswordReset`, `ChangePassword`, `RequestEmailChange`), plus `Config`, `Routes`, and
  `Close` in `plugin.go`, and `Routes` in `routes.go`.
- 81.0%: `Init` (`plugin.go`).

**`plugins/email-password/handlers`** (+21)

- `sign_in_handler.go` `Handler` (70.5%) and `sign_up_handler.go` `Handler` (71.0%).
- 78–86%: the change-password, email-change, password-reset, email-verification, and verify-email
  handlers.

**`core/events`** (+13)

- Watermill provider init functions: `initRedis` (32.0%), `initPostgres` (57.9%),
  `initRabbitMQ` (69.2%), `initNATS` (71.4%), `initKafka` (94.4%), `initSQLite` (81.0%).
- `watermill_pubsub.go`: `Subscribe` (80.0%) and `Close` (87.5%).
- Much of this is broker connection code (see Unreachable branches). Config and validation
  branches in these functions can still be tested.

**`openapi`** (+11)

- 0%: `WithJSONContentType` (`options.go`).
- Under 80%: `wrapNullableRefInAnyOf` (37.5%), `initV31` (73.3%), `NewOpenAPIService` (75.0%),
  `AddOperation` (75.0%), `RegisterSchema` (71.4%), `NewOpenAPISpecHandler` (71.4%),
  `NewOpenAPISpecYAMLHandler` (71.4%).

**`plugins/email-password/services`** (+8)

- `hooks_executor.go`: `AfterChangePassword` (28.6%), `BeforeRequestPasswordReset` (40.0%),
  `BeforeRequestEmailChange` (40.0%).

**`core/email/template`** (+4)

- `NewCommonContext` (0%), `Title` (66.7%, `funcs.go`), `Register` (77.3%), and `Execute` (84.6%)
  in `manager.go`.

**`plugins/otel/types`** (+1)

- `RuntimeOn` (0%), `LogsActive` (0%), and `boolValue` (66.7%) in `config.go`.

## Unreachable branches

These were documented by the slices that encountered them and are not excluded from coverage:

- **Brokers:** Watermill Postgres/Redis/Kafka/NATS/RabbitMQ success paths need live servers
  (`core/events/watermill_providers.go`).
- **SQL dialects:** Postgres/MySQL migration SQL cannot execute on SQLite; tests assert provider
  selection and statement contents instead (`migrations/core.go`, `plugins/totp/migrations.go`).
- **Database drivers:** MySQL `bun.NewDB` runs `SELECT version()` during initialization, so the
  MySQL success path needs a server (`internal/bootstrap/database.go`).
- **Defensive branches:** e.g. `RowsAffected` errors, `crypto/rand.Read` failures, and claim-setter
  type errors that the libraries never produce with the values used.
- **OTel constants:** `resource.Merge` cannot fail with the shared schema URL, and
  `runtime.Start` has no reachable registration error (`plugins/otel/plugin.go`).

## Optional next steps

1. Write unit tests for the gaps above, largest deficit first: `email-password/usecases`,
   `organizations/services`, then `email-password` and `email-password/handlers`.
2. Re-measure and test the `oauth2` packages after the OAuth 2.1 / OIDC rewrite lands.
3. Once coverage has been above 90% for a release, raise the Codecov gate in `codecov.yml` from
   80% to 90%. CI configuration is unchanged by this effort.
