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
