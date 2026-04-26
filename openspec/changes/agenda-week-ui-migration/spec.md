# Specification — agenda-week-ui-migration

## Requirement 1 — Single weekly agenda source
The schedule UI MUST use `/agenda/week` as the primary source for weekly agenda reads.

### Scenario 1.1 — Weekly board loads from composed agenda endpoint
**Given** an authenticated actor with access to a professional agenda
**When** the weekly board is loaded or refreshed
**Then** the frontend SHALL request `/agenda/week` with `professional_id` and `week_start`
**And** it SHALL stop recomposing the week through daily `listSlots` + `listAppointments` reads.

### Scenario 1.2 — Weekly board reflects composed agenda payload
**Given** `/agenda/week` returns templates, blocks, consultations and slots
**When** the frontend adapts the response
**Then** the board SHALL derive visible days, time bands and slot occupancy from that single payload
**And** it SHALL remain renderable even when consultations exist without `slot_id`.

## Requirement 2 — Domain adaptation boundary
The schedule UI MUST keep domain adaptation outside React rendering components.

### Scenario 2.1 — Adapter owns domain-to-UI mapping
**Given** a `WeekAgenda` payload
**When** the schedule screen prepares board state
**Then** a dedicated adapter SHALL transform backend entities into a render-ready board model
**And** `ScheduleDemo` SHALL consume that model instead of embedding transformation rules inline.

### Scenario 2.2 — Adapter handles consultation states safely
**Given** consultations can be `scheduled`, `checked_in`, `completed`, `cancelled` or `no_show`
**When** the adapter builds the board model
**Then** it SHALL map those states to deterministic visual states
**And** the first iteration MAY keep a simplified UI vocabulary as long as state meaning is not lost.

## Requirement 3 — Legacy compatibility during migration
The first iteration MUST preserve compatibility while the writing flow is still legacy.

### Scenario 3.1 — Reading migrates before writing
**Given** booking and cancellation still use legacy operations at the start of this change
**When** the weekly board migrates to `/agenda/week`
**Then** the reading path SHALL switch first
**And** writing/cancellation MAY remain on legacy endpoints temporarily.

### Scenario 3.2 — Legacy compatibility is explicit, not primary
**Given** `/appointments` still exists
**When** the schedule UI is migrated
**Then** `/appointments` SHALL be treated as compatibility surface
**And** new weekly board rendering SHALL NOT depend on legacy daily list assembly.

## Requirement 4 — Test coverage follows the new contract
Frontend tests MUST validate the new agenda contract and adapter behavior.

### Scenario 4.1 — Schedule screen tests use week agenda fixtures
**Given** the schedule screen test suite
**When** weekly loading is tested
**Then** tests SHALL mock `fetchWeekAgenda()` directly
**And** they SHALL stop fabricating the board through 10 legacy requests.

### Scenario 4.2 — Adapter tests cover mixed agenda cases
**Given** the adapter is introduced
**When** fixtures include slots, blocks, template versions and standalone consultations
**Then** the adapter SHALL be covered by focused tests
**And** those tests SHALL verify band generation, day summaries and occupancy mapping.
