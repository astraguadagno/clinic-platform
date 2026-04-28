# Apply Progress — weekly-schedule-template-flow-ui

## Scope

Same-date `effective_from` replacement for schedule template versions and the HTTP contract cleanup around `POST /schedules`.

## Mode

Strict TDD is enabled by `openspec/config.yaml` and a Go test runner exists. This artifact records the evidence available for the same-date replacement work without inventing history.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| Remove stale duplicate-date 409 handling from `POST /schedules` | `services/appointments-service/internal/http/server_test.go` | HTTP handler unit/integration via `httptest` | ✅ `go test ./internal/http -run TestCreateSchedulePostScenarios` passed before this fix | ✅ Updated the existing ErrConflict scenario to expect the stale duplicate-date 409 branch to be gone; focused run failed with `status = 409, want 500` | ✅ Removed the schedule-specific ErrConflict mapping in `server.go`; focused tests passed | ✅ Added a second HTTP-level flow posting the same `effective_from` twice and asserting both responses are `201 Created` with replacement semantics | ✅ Ran `gofmt`; focused tests passed after formatting |
| Existing repository-level same-date replacement implementation | `services/appointments-service/internal/appointments/repository_test.go`, `services/appointments-service/internal/appointments/repository_postgres_integration_test.go` | Repository unit + Postgres integration | ⚠️ Existing safety net from verification: focused repository tests had already passed | ⚠️ RED-first for earlier repository changes cannot be reconstructed from available artifacts | ✅ Existing verification reported repository same-date replacement tests passed | ✅ Existing tests cover SQL upsert shape and Postgres replacement behavior | ➖ No refactor in this apply pass |
| Existing UI copy/API contract updates | `web/src/features/schedule/WeeklySchedulePage.test.tsx`, `services/appointments-service/openapi/openapi.yaml` | Frontend component + contract documentation | ⚠️ Existing safety net from verification: focused Vitest/OpenAPI review had already passed | ⚠️ RED-first for earlier UI/OpenAPI changes cannot be reconstructed from available artifacts | ✅ Existing verification reported focused UI tests passed | ✅ UI copy and OpenAPI were checked against same-date replacement behavior | ➖ No refactor in this apply pass |

## Test Runs

1. `go test ./internal/http -run TestCreateSchedulePostScenarios` — passed before editing HTTP files (safety net).
2. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — failed before production change with `status = 409, want 500` for the stale ErrConflict mapping.
3. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — passed after removing the stale mapping.
4. `gofmt -w internal/http/server.go internal/http/server_test.go` — formatting only, not a build.
5. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — passed after formatting.

## Decisions and Notes

- `POST /schedules` same-date replacement is modeled by repository upsert success, not by returning `appointments.ErrConflict`.
- The old `schedule version already exists for effective date` response was message-specific stale behavior and was removed.
- Other HTTP handlers still map `appointments.ErrConflict` to `409 Conflict` where the domain operation can genuinely conflict, such as slot creation, appointment booking/cancellation, blocks, and consultations.
- Broader incomplete tasks in `tasks.md` remain incomplete because they are outside this same-date replacement scope.

## Remaining Risks

- Earlier same-date replacement changes predate this artifact; their RED-first sequence cannot be proven retroactively.
- This pass used focused HTTP tests only and did not run builds, per project/user constraint.
