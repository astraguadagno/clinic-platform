import { request } from './http';
import type { CreateEncounterPayload, Encounter, EncounterListResponse } from '../types/clinical';

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
