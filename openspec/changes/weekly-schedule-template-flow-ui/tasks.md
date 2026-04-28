# Tasks — weekly-schedule-template-flow-ui

## Phase 1 — Product/UX definition
- [ ] 1.1 Confirm the minimum first version of weekly editing: days, schedule windows, effective-from and preview.
- [ ] 1.2 Validate whether the first UI supports one or multiple time windows per day.
- [ ] 1.3 Align terminology in the UI: template, active schedule, future version, effective-from.
- [x] 1.4 Define the correction policy when a user needs to replace a version using the same `effective_from`.

## Phase 2 — Flow design
- [x] 2.1 Define the `Secretary` flow for selecting professional and editing weekly schedule.
- [x] 2.2 Define the `Doctor` flow scoped to own professional profile.
- [x] 2.3 Define where weekly editing lives relative to the existing weekly agenda board.

## Phase 3 — Preview and conflict handling
- [x] 3.1 Define the preview surface showing future agenda impact.
- [x] 3.2 Define minimum visible conflict information for future booked consultations.
- [ ] 3.3 Align with the separate backlog item for template conflict policy.

## Phase 4 — Version replacement and persistence policy
- [x] 4.1 Decide whether same-date correction is modeled as replace, edit-future-version, or delete+recreate.
- [x] 4.2 Align backend constraints/API behavior with the chosen same-date correction policy.
- [x] 4.3 Reflect that policy explicitly in UI copy and save flow.

## Phase 5 — Implementation handoff
- [x] 5.1 Translate the flow into a component map or wireframe-ready layout.
- [ ] 5.2 Link this change with the technical implementation sequence after `agenda-week-ui-migration`.
- [x] 5.3 Prepare a follow-up apply task once the UX/flow is approved.

## Backlog — Architecture follow-ups
- [ ] B.1 Refactor weekly schedule saving toward a cleaner application boundary: introduce a schedule template save use case, move professional-existence orchestration out of the HTTP handler, keep SQL-specific upsert behavior inside the repository, and rename persistence intent from `CreateTemplate` to `UpsertTemplateVersion`/equivalent so product language and storage language stop being conflated.
