import { request } from './http';
import type {
  ClinicalHistory,
  CreateEncounterPayload,
  CreatePatientClinicalNotePayload,
  Encounter,
  EncounterListResponse,
  PatientClinicalNote,
  PatientClinicalNoteListResponse,
  UpdateClinicalHistoryPayload,
  UpdatePatientClinicalNotePayload,
} from '../types/clinical';

const DIRECTORY_API_BASE = '/directory-api';

export function listPatientEncounters(patientId: string) {
  return request<EncounterListResponse>(DIRECTORY_API_BASE, `/patients/${patientId}/encounters`, {
    auth: true,
  });
}

export function createPatientEncounter(patientId: string, payload: CreateEncounterPayload) {
  return request<Encounter>(DIRECTORY_API_BASE, `/patients/${patientId}/encounters`, {
    method: 'POST',
    body: payload,
    auth: true,
  });
}

export function getClinicalHistory(patientId: string) {
  return request<ClinicalHistory>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-history`, {
    auth: true,
  });
}

export function updateClinicalHistory(patientId: string, payload: UpdateClinicalHistoryPayload) {
  return request<ClinicalHistory>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-history`, {
    method: 'PATCH',
    body: payload,
    auth: true,
  });
}

export function listPatientClinicalNotes(patientId: string) {
  return request<PatientClinicalNoteListResponse>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-notes`, {
    auth: true,
  });
}

export function createPatientClinicalNote(patientId: string, payload: CreatePatientClinicalNotePayload) {
  return request<PatientClinicalNote>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-notes`, {
    method: 'POST',
    body: payload,
    auth: true,
  });
}

export function updatePatientClinicalNote(patientId: string, noteId: string, payload: UpdatePatientClinicalNotePayload) {
  return request<PatientClinicalNote>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-notes/${noteId}`, {
    method: 'PATCH',
    body: payload,
    auth: true,
  });
}

export function deletePatientClinicalNote(patientId: string, noteId: string) {
  return request<unknown>(DIRECTORY_API_BASE, `/patients/${patientId}/clinical-notes/${noteId}`, {
    method: 'DELETE',
    auth: true,
  });
}
