import type { Patient } from '../../types/directory';

type SearchablePatient = Pick<Patient, 'first_name' | 'last_name' | 'document'>;

export function normalizePatientSearchValue(value: string) {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
    .replace(/\s+/g, ' ');
}

export function buildPatientSearchIndex(patient: SearchablePatient) {
  return [
    patient.first_name,
    patient.last_name,
    `${patient.first_name} ${patient.last_name}`,
    `${patient.last_name} ${patient.first_name}`,
    patient.document,
  ]
    .map(normalizePatientSearchValue)
    .filter(Boolean)
    .join(' ');
}

export function filterPatients<T extends SearchablePatient>(patients: T[], query: string) {
  const normalizedQuery = normalizePatientSearchValue(query);

  if (!normalizedQuery) {
    return patients;
  }

  return patients.filter((patient) => buildPatientSearchIndex(patient).includes(normalizedQuery));
}
