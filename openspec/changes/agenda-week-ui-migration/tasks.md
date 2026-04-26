# Tasks — agenda-week-ui-migration

## Phase 1 — Adapter boundary
- [ ] 1.1 Create `web/src/features/schedule/agendaAdapter.ts` with a typed `WeekAgenda -> board model` transformation.
- [ ] 1.2 Define a render-ready board model for days, time bands, cell state and summaries.
- [ ] 1.3 Add focused adapter tests in `web/src/features/schedule/agendaAdapter.test.ts` for slots, blocks and standalone consultations.

## Phase 2 — Weekly read migration
- [ ] 2.1 Update `web/src/api/appointments.ts` usage so `ScheduleDemo` reads `fetchWeekAgenda()` as the primary weekly source.
- [ ] 2.2 Refactor `ScheduleDemo.tsx` to consume the adapter output instead of `listOperationalWeek()`.
- [ ] 2.3 Keep existing UX behavior as stable as possible while changing the data source.

## Phase 3 — Test migration
- [ ] 3.1 Rewrite `web/src/features/schedule/ScheduleDemo.test.tsx` to mock `fetchWeekAgenda()` directly.
- [ ] 3.2 Keep only the minimum legacy API tests needed in `web/src/api/appointments.test.ts`.
- [ ] 3.3 Run targeted Vitest coverage for the new adapter and schedule screen.

## Phase 4 — Compatibility follow-up
- [ ] 4.1 Confirm that the weekly board no longer depends on legacy daily list assembly.
- [ ] 4.2 Document `/appointments` as temporary compatibility surface in code/comments or follow-up artifacts.
- [ ] 4.3 Prepare the next change for migrating booking/cancellation to `consultations`.
