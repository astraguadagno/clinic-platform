CREATE TABLE IF NOT EXISTS clinical_charts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    professional_id UUID NOT NULL REFERENCES professionals(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT clinical_charts_owner_unique UNIQUE (patient_id, professional_id)
);

CREATE INDEX IF NOT EXISTS clinical_charts_patient_id_idx ON clinical_charts (patient_id);
CREATE INDEX IF NOT EXISTS clinical_charts_professional_id_idx ON clinical_charts (professional_id);

CREATE TABLE IF NOT EXISTS clinical_encounters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chart_id UUID NOT NULL REFERENCES clinical_charts(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    professional_id UUID NOT NULL REFERENCES professionals(id) ON DELETE CASCADE,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS clinical_encounters_chart_id_idx ON clinical_encounters (chart_id);
CREATE INDEX IF NOT EXISTS clinical_encounters_patient_owner_idx ON clinical_encounters (patient_id, professional_id);
CREATE INDEX IF NOT EXISTS clinical_encounters_occurred_at_idx ON clinical_encounters (occurred_at DESC);

CREATE TABLE IF NOT EXISTS clinical_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chart_id UUID NOT NULL REFERENCES clinical_charts(id) ON DELETE CASCADE,
    encounter_id UUID NOT NULL REFERENCES clinical_encounters(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    professional_id UUID NOT NULL REFERENCES professionals(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (btrim(kind) <> ''),
    content TEXT NOT NULL CHECK (btrim(content) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT clinical_notes_encounter_kind_unique UNIQUE (encounter_id, kind)
);

CREATE INDEX IF NOT EXISTS clinical_notes_chart_id_idx ON clinical_notes (chart_id);
CREATE INDEX IF NOT EXISTS clinical_notes_patient_owner_idx ON clinical_notes (patient_id, professional_id);
