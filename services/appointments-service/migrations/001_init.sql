CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

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
    CONSTRAINT availability_slots_professional_start_unique UNIQUE (professional_id, start_time),
    CONSTRAINT availability_slots_no_overlap EXCLUDE USING gist (
        professional_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    )
);

CREATE INDEX availability_slots_professional_start_idx
    ON availability_slots (professional_id, start_time);
CREATE INDEX availability_slots_status_idx
    ON availability_slots (status);

CREATE TABLE consultations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id UUID,
    professional_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'scheduled',
    source TEXT NOT NULL DEFAULT 'secretary',
    notes TEXT,
    check_in_time TIMESTAMPTZ,
    reception_notes TEXT,
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT consultations_slot_fk FOREIGN KEY (slot_id) REFERENCES availability_slots(id),
    CONSTRAINT consultations_status_valid CHECK (status IN ('scheduled', 'checked_in', 'completed', 'cancelled', 'no_show')),
    CONSTRAINT consultations_cancelled_at_consistency CHECK (
        (status = 'cancelled' AND cancelled_at IS NOT NULL) OR
        (status IN ('scheduled', 'checked_in', 'completed', 'no_show') AND cancelled_at IS NULL)
    ),
    CONSTRAINT consultations_source_valid CHECK (source IN ('online', 'secretary', 'doctor')),
    CONSTRAINT consultations_scheduled_range_valid CHECK (scheduled_start < scheduled_end)
);

CREATE UNIQUE INDEX consultations_slot_scheduled_unique_idx
    ON consultations (slot_id)
    WHERE status = 'scheduled';
CREATE INDEX consultations_patient_id_idx
    ON consultations (patient_id);
CREATE INDEX consultations_professional_id_idx
    ON consultations (professional_id);
CREATE INDEX consultations_status_idx
    ON consultations (status);
CREATE INDEX consultations_professional_scheduled_start_idx
    ON consultations (professional_id, scheduled_start);

CREATE TABLE schedule_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professional_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_templates_professional_unique UNIQUE (professional_id)
);

CREATE INDEX schedule_templates_professional_idx
    ON schedule_templates (professional_id);

CREATE TABLE schedule_template_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES schedule_templates(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    effective_from DATE NOT NULL,
    recurrence JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    reason TEXT,
    CONSTRAINT schedule_template_versions_version_number_positive CHECK (version_number > 0),
    CONSTRAINT schedule_template_versions_effective_from_unique UNIQUE (template_id, effective_from),
    CONSTRAINT schedule_template_versions_version_unique UNIQUE (template_id, version_number),
    CONSTRAINT schedule_template_versions_recurrence_object CHECK (jsonb_typeof(recurrence) = 'object')
);

CREATE INDEX schedule_template_versions_template_idx
    ON schedule_template_versions (template_id);
CREATE INDEX schedule_template_versions_template_effective_from_idx
    ON schedule_template_versions (template_id, effective_from DESC);
CREATE INDEX schedule_template_versions_effective_from_idx
    ON schedule_template_versions (effective_from);

CREATE TABLE schedule_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professional_id UUID NOT NULL,
    scope TEXT NOT NULL,
    block_date DATE,
    start_date DATE,
    end_date DATE,
    day_of_week SMALLINT,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    template_id UUID REFERENCES schedule_templates(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_blocks_scope_valid CHECK (scope IN ('single', 'range', 'template')),
    CONSTRAINT schedule_blocks_time_range_valid CHECK (start_time < end_time),
    CONSTRAINT schedule_blocks_day_of_week_valid CHECK (day_of_week IS NULL OR day_of_week BETWEEN 1 AND 7),
    CONSTRAINT schedule_blocks_scope_columns_valid CHECK (
        (scope = 'single' AND block_date IS NOT NULL AND start_date IS NULL AND end_date IS NULL AND day_of_week IS NULL AND template_id IS NULL) OR
        (scope = 'range' AND block_date IS NULL AND start_date IS NOT NULL AND end_date IS NOT NULL AND start_date <= end_date AND day_of_week IS NULL AND template_id IS NULL) OR
        (scope = 'template' AND block_date IS NULL AND start_date IS NULL AND end_date IS NULL AND day_of_week IS NOT NULL AND template_id IS NOT NULL)
    )
);

CREATE INDEX schedule_blocks_professional_scope_idx
    ON schedule_blocks (professional_id, scope);
CREATE INDEX schedule_blocks_professional_block_date_idx
    ON schedule_blocks (professional_id, block_date);
CREATE INDEX schedule_blocks_professional_range_idx
    ON schedule_blocks (professional_id, start_date, end_date);
CREATE INDEX schedule_blocks_template_idx
    ON schedule_blocks (template_id);
