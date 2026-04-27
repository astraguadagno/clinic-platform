ALTER TABLE consultations
    ADD COLUMN scheduled_start TIMESTAMPTZ,
    ADD COLUMN scheduled_end TIMESTAMPTZ;

UPDATE consultations AS consultations_to_backfill
SET scheduled_start = availability_slots.start_time,
    scheduled_end = availability_slots.end_time
FROM availability_slots
WHERE consultations_to_backfill.slot_id = availability_slots.id
  AND (consultations_to_backfill.scheduled_start IS NULL OR consultations_to_backfill.scheduled_end IS NULL);

ALTER TABLE consultations
    ALTER COLUMN scheduled_start SET NOT NULL,
    ALTER COLUMN scheduled_end SET NOT NULL,
    ADD CONSTRAINT consultations_scheduled_range_valid CHECK (scheduled_start < scheduled_end);

CREATE INDEX consultations_professional_scheduled_start_idx
    ON consultations (professional_id, scheduled_start);
