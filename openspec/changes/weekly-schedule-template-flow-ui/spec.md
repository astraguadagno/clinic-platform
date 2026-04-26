# Specification — weekly-schedule-template-flow-ui

## Requirement 1 — Weekly schedule editing surface
The product MUST provide a dedicated flow to create or edit a professional weekly schedule template.

### Scenario 1.1 — Enter weekly template editing mode
**Given** a user with permission to manage schedules
**When** they access weekly schedule editing for a professional
**Then** the UI SHALL present a dedicated editing surface for the weekly template
**And** it SHALL NOT require the user to infer the template only from daily booking interactions.

### Scenario 1.2 — Weekly template editing supports day-based configuration
**Given** the weekly template editor is open
**When** the user configures the schedule
**Then** the UI SHALL allow enabling or disabling days of the week
**And** it SHALL allow defining schedule windows for active days
**And** it SHALL make clear which changes belong to the template, not to individual bookings.

## Requirement 2 — Effective-from versioning must be explicit
The UI MUST make `effective_from` part of the main decision flow, not a hidden technical field.

### Scenario 2.1 — User chooses when the new template becomes active
**Given** a user is saving a new weekly template version
**When** they confirm the change
**Then** the flow SHALL require an `effective_from` value
**And** the UI SHALL explain that the new version affects future agenda from that date onward.

### Scenario 2.2 — Current vs future template versions are distinguishable
**Given** a professional already has an active template
**When** the user prepares a new version
**Then** the UI SHALL distinguish the current active schedule from the future schedule version being saved.

### Scenario 2.3 — User can correct a same-day scheduling mistake without inventing a different date
**Given** a user has just created or prepared a weekly template version for a given `effective_from`
**When** they detect a mistake and try to save the corrected version with that same `effective_from`
**Then** the product SHALL provide an explicit correction path
**And** it SHALL NOT force the user to choose an artificial different date only to bypass a uniqueness constraint.

### Scenario 2.4 — Same-date replacement behavior is explicit
**Given** a version already exists for the requested `effective_from`
**When** the user attempts to save another version for that same date
**Then** the system SHALL either allow explicit replacement or expose a controlled edit/replace flow
**And** the UX SHALL explain what happens to the previously stored future version.

## Requirement 3 — Preview before commit
The user MUST be able to preview the impact of the weekly template before confirming it.

### Scenario 3.1 — Preview shows resulting future agenda shape
**Given** a user modifies the weekly template and selects `effective_from`
**When** they request a preview or reach the confirmation step
**Then** the UI SHALL present the resulting future agenda shape
**And** it SHALL help the user understand what availability changes after the effective date.

### Scenario 3.2 — Preview includes known future conflicts
**Given** future consultations or bookings may conflict with the new template
**When** the preview is generated
**Then** the UI SHALL surface detected conflicts
**And** it SHALL identify affected dates or time windows before the user commits the change.

## Requirement 4 — Actor-aware UX
The flow MUST adapt to the actor operating it.

### Scenario 4.1 — Secretary operates multi-doctor schedule editing
**Given** a `Secretary` user
**When** they enter weekly schedule editing
**Then** the flow SHALL allow selecting the target professional
**And** it SHALL support multi-doctor operational context without implying ownership restrictions that do not apply to secretariat work.

### Scenario 4.2 — Doctor edits only own weekly schedule
**Given** a `Doctor` user linked to one professional profile
**When** they enter weekly schedule editing
**Then** the flow SHALL be scoped to their own professional agenda
**And** it SHALL NOT expose controls suggesting they can edit other professionals.

## Requirement 5 — Editing flow stays separate from daily operations
Weekly template editing MUST remain conceptually separate from daily agenda booking and cancellation.

### Scenario 5.1 — Template editing is not embedded into booking interactions
**Given** the weekly agenda board also supports daily operational actions
**When** weekly template editing is introduced
**Then** the UI SHALL keep template editing in a dedicated mode, panel or surface
**And** it SHALL avoid mixing template definition directly into day-by-day booking controls.

### Scenario 5.2 — Future implementation can map to concrete components
**Given** this flow will later be implemented in React
**When** engineers break it into tasks
**Then** the specification SHALL support decomposition into clear UI sections such as professional context, template editor, effective date, preview and conflict summary.
