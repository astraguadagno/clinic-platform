-- Schedule blocks subtract availability from recurring templates or specific dates.
-- Scope defines which date columns are valid for each block type.
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
