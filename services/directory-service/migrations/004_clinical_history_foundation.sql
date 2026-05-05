CREATE TABLE IF NOT EXISTS clinical_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    weight_kg NUMERIC(5,2) CHECK (weight_kg IS NULL OR (weight_kg > 0 AND weight_kg <= 500)),
    height_cm NUMERIC(5,2) CHECK (height_cm IS NULL OR (height_cm > 0 AND height_cm <= 300)),
    antecedentes TEXT CHECK (antecedentes IS NULL OR char_length(antecedentes) <= 4000),
    allergies TEXT CHECK (allergies IS NULL OR char_length(allergies) <= 4000),
    habitual_medication TEXT CHECK (habitual_medication IS NULL OR char_length(habitual_medication) <= 4000),
    chronic_conditions TEXT CHECK (chronic_conditions IS NULL OR char_length(chronic_conditions) <= 4000),
    habits TEXT CHECK (habits IS NULL OR char_length(habits) <= 4000),
    general_observations TEXT CHECK (general_observations IS NULL OR char_length(general_observations) <= 4000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT clinical_history_patient_unique UNIQUE (patient_id)
);

CREATE INDEX IF NOT EXISTS clinical_history_patient_id_idx ON clinical_history (patient_id);
