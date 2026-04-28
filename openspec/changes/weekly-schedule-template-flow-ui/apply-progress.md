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
| Book generated template slots from agenda | `web/src/features/schedule/ScheduleDemo.test.tsx`, `web/src/features/schedule/agendaAdapter.test.ts`, `web/src/api/appointments.test.ts` | Frontend component + adapter/API client | ✅ Existing click-selection tests passed, but did not cover virtual template slots with empty backend IDs | ✅ Added focused test for generated template slots; without a synthetic slot identity and `/consultations` booking path this flow cannot reserve by persisted `slot_id` | ✅ Added stable synthetic IDs for virtual template slots and route their booking through `POST /consultations` with `scheduled_start`/`scheduled_end`; focused tests passed | ✅ Docker API smoke confirmed standalone consultation creation by scheduled range returns `201` after local schema drift repair | ✅ Kept legacy persisted-slot booking on `/appointments`; generated slots use the consultation path only |
| Reconcile local Docker appointments schema drift | `deploy/docker-compose.yml`, `services/appointments-service/migrations/008_reconcile_local_schema.sql` | Local-dev infrastructure | ✅ Real Docker smoke exposed schema drift in existing appointments volumes | ✅ Existing volume was missing required consultation shape until manually repaired; compose had no appointments migrator | ✅ Added an idempotent `appointments-db-migrator` that applies defensive reconciliation SQL before `appointments-service` starts | ✅ `docker compose -f deploy/docker-compose.yml config` passed; `docker compose -f deploy/docker-compose.yml up appointments-db-migrator` completed with exit code 0 against current volume | ⚠️ Pragmatic debt fix only; not a full migration framework |

## Test Runs

1. `go test ./internal/http -run TestCreateSchedulePostScenarios` — passed before editing HTTP files (safety net).
2. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — failed before production change with `status = 409, want 500` for the stale ErrConflict mapping.
3. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — passed after removing the stale mapping.
4. `gofmt -w internal/http/server.go internal/http/server_test.go` — formatting only, not a build.
5. `go test ./internal/http -run 'TestCreateSchedulePost(Scenarios|ReplacesSameEffectiveFromWithoutConflict)'` — passed after formatting.
6. `npm test -- src/features/schedule/ScheduleDemo.test.tsx src/features/schedule/agendaAdapter.test.ts src/api/appointments.test.ts` — passed after generated-slot booking fix (`18 passed`).
7. Docker API smoke: `POST /consultations` with `scheduled_start`/`scheduled_end` returned `201`; `GET /agenda/week` returned `200` with the created consultation visible. This required repairing local Docker schema drift (`consultations.slot_id DROP NOT NULL`) caused by an old volume.
8. `docker compose -f deploy/docker-compose.yml config` — passed after adding `appointments-db-migrator`.
9. `docker compose -f deploy/docker-compose.yml up appointments-db-migrator` — completed successfully against the current local volume and showed only expected `IF NOT EXISTS` notices.

## Decisions and Notes

- `POST /schedules` same-date replacement is modeled by repository upsert success, not by returning `appointments.ErrConflict`.
- The old `schedule version already exists for effective date` response was message-specific stale behavior and was removed.
- Other HTTP handlers still map `appointments.ErrConflict` to `409 Conflict` where the domain operation can genuinely conflict, such as slot creation, appointment booking/cancellation, blocks, and consultations.
- Broader incomplete tasks in `tasks.md` remain incomplete because they are outside this same-date replacement scope.
- Agenda slots generated from weekly templates are virtual availability and do not have persisted `availability_slots.id`; booking those slots must create consultations by scheduled range instead of using the legacy appointment-by-slot path.
- `008_reconcile_local_schema.sql` intentionally fails if existing consultations still lack `scheduled_start`/`scheduled_end` after slot-based backfill, because inventing appointment times would corrupt data.

## Remaining Risks

- Earlier same-date replacement changes predate this artifact; their RED-first sequence cannot be proven retroactively.
- This pass used focused HTTP tests only and did not run builds, per project/user constraint.
- Existing Docker volumes can drift from current migrations because compose initialization does not replay all appointments migrations on already-created volumes; the new reconciliation migrator is a pragmatic local-dev fix, not a full migration framework.
