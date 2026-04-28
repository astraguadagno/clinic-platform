-- Pragmatic local Docker reconciliation for appointments DB schemas that predate
-- the current consultations/schedule shape. This file is intentionally
-- idempotent and safe to run on existing local volumes; it is not a general
-- migration framework.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE consultations
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'secretary',
    ADD COLUMN IF NOT EXISTS notes TEXT,
    ADD COLUMN IF NOT EXISTS check_in_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reception_notes TEXT,
    ADD COLUMN IF NOT EXISTS scheduled_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS scheduled_end TIMESTAMPTZ;

ALTER TABLE consultations
    ALTER COLUMN slot_id DROP NOT NULL;

UPDATE consultations AS consultations_to_backfill
SET scheduled_start = availability_slots.start_time,
    scheduled_end = availability_slots.end_time
FROM availability_slots
WHERE consultations_to_backfill.slot_id = availability_slots.id
  AND (consultations_to_backfill.scheduled_start IS NULL OR consultations_to_backfill.scheduled_end IS NULL);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM consultations
        WHERE scheduled_start IS NULL OR scheduled_end IS NULL
    ) THEN
        RAISE EXCEPTION
            'Cannot reconcile consultations scheduled range: scheduled_start/scheduled_end remain NULL after slot backfill. Fix or remove orphaned local rows before starting appointments-service.';
    END IF;

    ALTER TABLE consultations
        ALTER COLUMN scheduled_start SET NOT NULL,
        ALTER COLUMN scheduled_end SET NOT NULL;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'consultations'::regclass
          AND conname = 'consultations_scheduled_range_valid'
    ) THEN
        ALTER TABLE consultations
            ADD CONSTRAINT consultations_scheduled_range_valid CHECK (scheduled_start < scheduled_end);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS consultations_professional_scheduled_start_idx
    ON consultations (professional_id, scheduled_start);

CREATE TABLE IF NOT EXISTS schedule_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professional_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_templates_professional_unique UNIQUE (professional_id)
);

CREATE INDEX IF NOT EXISTS schedule_templates_professional_idx
    ON schedule_templates (professional_id);

CREATE TABLE IF NOT EXISTS schedule_template_versions (
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

CREATE INDEX IF NOT EXISTS schedule_template_versions_template_idx
    ON schedule_template_versions (template_id);
CREATE INDEX IF NOT EXISTS schedule_template_versions_template_effective_from_idx
    ON schedule_template_versions (template_id, effective_from DESC);
CREATE INDEX IF NOT EXISTS schedule_template_versions_effective_from_idx
    ON schedule_template_versions (effective_from);

CREATE TABLE IF NOT EXISTS schedule_blocks (
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

CREATE INDEX IF NOT EXISTS schedule_blocks_professional_scope_idx
    ON schedule_blocks (professional_id, scope);
CREATE INDEX IF NOT EXISTS schedule_blocks_professional_block_date_idx
    ON schedule_blocks (professional_id, block_date);
CREATE INDEX IF NOT EXISTS schedule_blocks_professional_range_idx
    ON schedule_blocks (professional_id, start_date, end_date);
CREATE INDEX IF NOT EXISTS schedule_blocks_template_idx
    ON schedule_blocks (template_id);
