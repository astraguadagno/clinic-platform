-- Consultation entity evolution: migrate the legacy appointments table into
-- consultations, align the status taxonomy, and add consultation metadata.

ALTER TABLE appointments RENAME TO consultations;

ALTER TABLE consultations
    RENAME CONSTRAINT appointments_pkey TO consultations_pkey;

ALTER TABLE consultations
    RENAME CONSTRAINT appointments_slot_fk TO consultations_slot_fk;

ALTER TABLE consultations
    ADD COLUMN source TEXT NOT NULL DEFAULT 'secretary',
    ADD COLUMN notes TEXT;

ALTER TABLE consultations
    DROP CONSTRAINT IF EXISTS appointments_status_valid,
    DROP CONSTRAINT IF EXISTS appointments_cancelled_at_consistency;

UPDATE consultations
SET status = 'scheduled'
WHERE status = 'booked';

ALTER TABLE consultations
    ALTER COLUMN status SET DEFAULT 'scheduled';

ALTER TABLE consultations
    ADD CONSTRAINT consultations_status_valid CHECK (status IN ('scheduled', 'checked_in', 'completed', 'cancelled', 'no_show')),
    ADD CONSTRAINT consultations_cancelled_at_consistency CHECK (
        (status = 'cancelled' AND cancelled_at IS NOT NULL) OR
        (status IN ('scheduled', 'checked_in', 'completed', 'no_show') AND cancelled_at IS NULL)
    ),
    ADD CONSTRAINT consultations_source_valid CHECK (source IN ('online', 'secretary', 'doctor'));

DROP INDEX IF EXISTS appointments_slot_booked_unique_idx;

CREATE UNIQUE INDEX consultations_slot_scheduled_unique_idx
    ON consultations (slot_id)
    WHERE status = 'scheduled';

ALTER INDEX IF EXISTS appointments_patient_id_idx
    RENAME TO consultations_patient_id_idx;

ALTER INDEX IF EXISTS appointments_professional_id_idx
    RENAME TO consultations_professional_id_idx;

ALTER INDEX IF EXISTS appointments_status_idx
    RENAME TO consultations_status_idx;
