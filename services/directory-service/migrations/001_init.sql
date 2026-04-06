CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE patients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL CHECK (btrim(first_name) <> ''),
    last_name TEXT NOT NULL CHECK (btrim(last_name) <> ''),
    document TEXT NOT NULL CHECK (btrim(document) <> ''),
    birth_date DATE NOT NULL,
    phone TEXT NOT NULL CHECK (btrim(phone) <> ''),
    email TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT patients_document_unique UNIQUE (document),
    CONSTRAINT patients_email_not_blank CHECK (email IS NULL OR btrim(email) <> '')
);

CREATE INDEX patients_last_name_idx ON patients (last_name);

CREATE TABLE professionals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL CHECK (btrim(first_name) <> ''),
    last_name TEXT NOT NULL CHECK (btrim(last_name) <> ''),
    specialty TEXT NOT NULL CHECK (btrim(specialty) <> ''),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX professionals_last_name_idx ON professionals (last_name);
CREATE INDEX professionals_specialty_idx ON professionals (specialty);
