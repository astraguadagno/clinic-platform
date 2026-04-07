ALTER TABLE appointments
    DROP CONSTRAINT IF EXISTS appointments_slot_unique;

CREATE UNIQUE INDEX IF NOT EXISTS appointments_slot_booked_unique_idx
    ON appointments (slot_id)
    WHERE status = 'booked';
