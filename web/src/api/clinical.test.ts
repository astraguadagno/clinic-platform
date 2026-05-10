import { beforeEach, describe, expect, it, vi } from 'vitest';

const requestMock = vi.fn();

vi.mock('./http', () => ({
  request: requestMock,
}));

describe('clinical API patient history and standalone notes', () => {
  beforeEach(() => {
    requestMock.mockReset();
  });

  it('uses authenticated directory routes for clinical history reads and writes', async () => {
    requestMock.mockResolvedValue({ id: 'history-1', patient_id: 'patient-1' });

    const clinical = await import('./clinical');

    await clinical.getClinicalHistory('patient-1');
    await clinical.updateClinicalHistory('patient-1', {
      allergies: 'Penicilina',
      habitual_medication: null,
      weight_kg: 72.5,
    });

    expect(requestMock).toHaveBeenNthCalledWith(1, '/directory-api', '/patients/patient-1/clinical-history', {
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(2, '/directory-api', '/patients/patient-1/clinical-history', {
      method: 'PATCH',
      body: {
        allergies: 'Penicilina',
        habitual_medication: null,
        weight_kg: 72.5,
      },
      auth: true,
    });
  });

  it('uses authenticated standalone note CRUD routes and preserves nullable consultation references', async () => {
    requestMock.mockResolvedValue({ items: [] });

    const clinical = await import('./clinical');

    await clinical.listPatientClinicalNotes('patient-1');
    await clinical.createPatientClinicalNote('patient-1', { content: 'Nota suelta', consultation_id: null });
    await clinical.updatePatientClinicalNote('patient-1', 'note-1', { content: 'Nota editada', consultation_id: 'consultation-1' });
    await clinical.deletePatientClinicalNote('patient-1', 'note-1');

    expect(requestMock).toHaveBeenNthCalledWith(1, '/directory-api', '/patients/patient-1/clinical-notes', {
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(2, '/directory-api', '/patients/patient-1/clinical-notes', {
      method: 'POST',
      body: { content: 'Nota suelta', consultation_id: null },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(3, '/directory-api', '/patients/patient-1/clinical-notes/note-1', {
      method: 'PATCH',
      body: { content: 'Nota editada', consultation_id: 'consultation-1' },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(4, '/directory-api', '/patients/patient-1/clinical-notes/note-1', {
      method: 'DELETE',
      auth: true,
    });
  });
});
