import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { PatientsWorkspace } from './PatientsWorkspace';

const { listPatientsMock, listPatientEncountersMock, createPatientEncounterMock } = vi.hoisted(() => ({
  listPatientsMock: vi.fn(),
  listPatientEncountersMock: vi.fn(),
  createPatientEncounterMock: vi.fn(),
}));

vi.mock('../../api/directory', () => ({
  listPatients: listPatientsMock,
}));

vi.mock('../../api/clinical', () => ({
  listPatientEncounters: listPatientEncountersMock,
  createPatientEncounter: createPatientEncounterMock,
}));

describe('PatientsWorkspace', () => {
  beforeEach(() => {
    listPatientsMock.mockReset();
    listPatientEncountersMock.mockReset();
    createPatientEncounterMock.mockReset();
  });

  it('shows doctor clinical data when mode allows encounters', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/modo clínico habilitado/i)).toBeInTheDocument();
    expect(await screen.findByText('Control inicial')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /guardar nota/i })).toBeEnabled();
  });

  it('keeps secretary in operational patient flow while denying encounters', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/modo operativo sin clinical encounters/i)).toBeInTheDocument();
    expect(screen.getByText(/encounters clínicos bloqueados/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /guardar nota/i })).toBeDisabled();
    expect(listPatientEncountersMock).not.toHaveBeenCalled();
  });

  it('keeps malformed doctor sessions active while showing the forbidden patients state', () => {
    const onSessionInvalid = vi.fn();

    render(
      <PatientsWorkspace
        patientsMode={{ kind: 'forbidden', message: 'Tu usuario doctor no tiene professional_id asociado.' }}
        onSessionInvalid={onSessionInvalid}
      />,
    );

    expect(screen.getByText(/pacientes bloqueado/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /acceso denegado/i })).toBeInTheDocument();
    expect(screen.getByText(/no tiene professional_id asociado/i)).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
    expect(listPatientsMock).not.toHaveBeenCalled();
    expect(listPatientEncountersMock).not.toHaveBeenCalled();
  });

  it('invalidates the session on 401 patient bootstrap failures', async () => {
    listPatientsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    const onSessionInvalid = vi.fn();

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });
  });

  it('keeps the session active on 403 encounter failures', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockRejectedValue(new ApiError('No podés ver encounters.', 403));
    const onSessionInvalid = vi.fn();

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={onSessionInvalid} />);

    expect(await screen.findByText('No podés ver encounters.')).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
  });
});

function activePatient() {
  return {
    id: 'patient-1',
    first_name: 'Juan',
    last_name: 'Pérez',
    document: '12345678',
    birth_date: '1990-01-01',
    phone: '555-1234',
    email: 'juan@example.com',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function encounter() {
  return {
    id: 'encounter-1',
    chart_id: 'chart-1',
    patient_id: 'patient-1',
    professional_id: 'professional-1',
    occurred_at: '2026-01-02T10:00:00Z',
    created_at: '2026-01-02T10:00:00Z',
    updated_at: '2026-01-02T10:00:00Z',
    initial_note: {
      id: 'note-1',
      encounter_id: 'encounter-1',
      chart_id: 'chart-1',
      patient_id: 'patient-1',
      professional_id: 'professional-1',
      kind: 'summary',
      content: 'Control inicial',
      created_at: '2026-01-02T10:00:00Z',
      updated_at: '2026-01-02T10:00:00Z',
    },
  };
}
