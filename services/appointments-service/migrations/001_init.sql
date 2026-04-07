CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE availability_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professional_id UUID NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT availability_slots_time_range_valid CHECK (start_time < end_time),
    CONSTRAINT availability_slots_status_valid CHECK (status IN ('available', 'booked', 'cancelled')),
    CONSTRAINT availability_slots_professional_start_unique UNIQUE (professional_id, start_time)
);

CREATE INDEX availability_slots_professional_start_idx
    ON availability_slots (professional_id, start_time);
CREATE INDEX availability_slots_status_idx ON availability_slots (status);

CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id UUID NOT NULL,
    professional_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'booked',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT appointments_status_valid CHECK (status IN ('booked', 'cancelled')),
    CONSTRAINT appointments_cancelled_at_consistency CHECK (
        (status = 'booked' AND cancelled_at IS NULL) OR
        (status = 'cancelled' AND cancelled_at IS NOT NULL)
    ),
    CONSTRAINT appointments_slot_fk FOREIGN KEY (slot_id) REFERENCES availability_slots(id)
);

CREATE UNIQUE INDEX appointments_slot_booked_unique_idx
    ON appointments (slot_id)
    WHERE status = 'booked';
CREATE INDEX appointments_patient_id_idx ON appointments (patient_id);
CREATE INDEX appointments_professional_id_idx ON appointments (professional_id);
CREATE INDEX appointments_status_idx ON appointments (status);
