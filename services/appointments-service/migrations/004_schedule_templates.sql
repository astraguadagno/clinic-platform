-- Schedule templates: one template family per professional.
-- Effective-dated history lives in schedule_template_versions.
CREATE TABLE schedule_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    professional_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedule_templates_professional_unique UNIQUE (professional_id)
);

CREATE INDEX schedule_templates_professional_idx
    ON schedule_templates (professional_id);

-- Schedule template versions: immutable snapshots keyed by effective_from.
-- recurrence stores the weekly pattern as JSONB so future per-weekday windows can
-- evolve without schema churn.
-- NOTE: deeper recurrence-shape validation and version sequencing belong in the
-- repository/service layer when write flows are implemented.
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
