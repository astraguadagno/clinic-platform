import type { ListResponse, Patient } from './directory';

export type { Patient };

export type ClinicalNote = {
  id: string;
  encounter_id: string;
  chart_id: string;
  patient_id: string;
  professional_id: string;
  kind: string;
  content: string;
  created_at: string;
  updated_at: string;
};

export type Encounter = {
  id: string;
  chart_id: string;
  patient_id: string;
  professional_id: string;
  occurred_at: string;
  created_at: string;
  updated_at: string;
  initial_note: ClinicalNote;
};

export type EncounterListResponse = ListResponse<Encounter>;

export type CreateEncounterPayload = {
  note: string;
  occurred_at?: string;
};

export type ClinicalHistory = {
  id: string;
  patient_id: string;
  weight_kg: number | null;
  height_cm: number | null;
  antecedentes: string | null;
  allergies: string | null;
  habitual_medication: string | null;
  chronic_conditions: string | null;
  habits: string | null;
  general_observations: string | null;
  created_at: string;
  updated_at: string;
};

export type UpdateClinicalHistoryPayload = Partial<{
  weight_kg: number | null;
  height_cm: number | null;
  antecedentes: string | null;
  allergies: string | null;
  habitual_medication: string | null;
  chronic_conditions: string | null;
  habits: string | null;
  general_observations: string | null;
}>;

export type PatientClinicalNote = {
  id: string;
  patient_id: string;
  professional_id: string;
  consultation_id?: string | null;
  kind: string;
  content: string;
  created_at: string;
  updated_at: string;
};

export type PatientClinicalNoteListResponse = ListResponse<PatientClinicalNote>;

export type CreatePatientClinicalNotePayload = {
  content: string;
  consultation_id?: string | null;
};

export type UpdatePatientClinicalNotePayload = Partial<CreatePatientClinicalNotePayload>;
