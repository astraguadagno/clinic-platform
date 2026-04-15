import { describe, expect, it } from 'vitest';
import { buildPatientSearchIndex, filterPatients, normalizePatientSearchValue } from './matching';

describe('patient search matching', () => {
  it('normalizes accents and collapses whitespace', () => {
    expect(normalizePatientSearchValue('  María   Pérez  ')).toBe('maria perez');
  });

  it('indexes full-name permutations and documents', () => {
    const patient = makePatient({ first_name: 'Ana María', last_name: 'Gómez', document: '20-12345678-9' });

    const index = buildPatientSearchIndex(patient);

    expect(index).toContain('ana maria');
    expect(index).toContain('gomez');
    expect(index).toContain('ana maria gomez');
    expect(index).toContain('gomez ana maria');
    expect(index).toContain('20-12345678-9');
  });

  it('matches by name permutation and document substring', () => {
    const patients = [
      makePatient({ id: 'patient-1', first_name: 'Ana María', last_name: 'Gómez', document: '20123456' }),
      makePatient({ id: 'patient-2', first_name: 'Juan', last_name: 'Pérez', document: '30999888' }),
    ];

    expect(filterPatients(patients, 'maria gomez').map((patient) => patient.id)).toEqual(['patient-1']);
    expect(filterPatients(patients, 'gomez ana').map((patient) => patient.id)).toEqual(['patient-1']);
    expect(filterPatients(patients, '9998').map((patient) => patient.id)).toEqual(['patient-2']);
  });

  it('returns all patients when query is empty', () => {
    const patients = [makePatient({ id: 'patient-1' }), makePatient({ id: 'patient-2' })];

    expect(filterPatients(patients, '')).toEqual(patients);
    expect(filterPatients(patients, '   ')).toEqual(patients);
  });
});

function makePatient(overrides: Partial<{ id: string; first_name: string; last_name: string; document: string }> = {}) {
  return {
    id: overrides.id ?? 'patient-1',
    first_name: overrides.first_name ?? 'Juan',
    last_name: overrides.last_name ?? 'Pérez',
    document: overrides.document ?? '12345678',
  };
}
