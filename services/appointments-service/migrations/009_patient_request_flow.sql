-- Patient request flow: allow patient-origin, request-first consultations.

ALTER TABLE consultations
    DROP CONSTRAINT IF EXISTS consultations_status_valid,
    DROP CONSTRAINT IF EXISTS consultations_source_valid,
    DROP CONSTRAINT IF EXISTS consultations_cancelled_at_consistency;

ALTER TABLE consultations
    ADD CONSTRAINT consultations_status_valid CHECK (status IN ('scheduled', 'requested', 'checked_in', 'completed', 'cancelled', 'no_show')),
    ADD CONSTRAINT consultations_cancelled_at_consistency CHECK (
        (status = 'cancelled' AND cancelled_at IS NOT NULL) OR
        (status IN ('scheduled', 'requested', 'checked_in', 'completed', 'no_show') AND cancelled_at IS NULL)
    ),
    ADD CONSTRAINT consultations_source_valid CHECK (source IN ('online', 'secretary', 'doctor', 'patient'));
