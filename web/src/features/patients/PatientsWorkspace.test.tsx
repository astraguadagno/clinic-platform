import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
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
    expect(screen.queryByRole('button', { name: /guardar nota/i })).not.toBeInTheDocument();
    expect(screen.getByText(/trabajo clínico oculto para secretaría/i)).toBeInTheDocument();
    expect(listPatientEncountersMock).not.toHaveBeenCalled();
  });

  it('offers directory support to secretary when no active patients exist', async () => {
    listPatientsMock.mockResolvedValue({ items: [] });
    const onOpenDirectorySupport = vi.fn();

    render(
      <PatientsWorkspace
        patientsMode={{ kind: 'secretary-operational' }}
        onSessionInvalid={vi.fn()}
        onOpenDirectorySupport={onOpenDirectorySupport}
      />,
    );

    expect(await screen.findByText(/no hay pacientes activos/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /abrir soporte de directorio/i }));

    expect(onOpenDirectorySupport).toHaveBeenCalledTimes(1);
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

  it('filters the sidebar while retaining the selected patient outside visible results and restores the list when cleared', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchInput = await screen.findByLabelText('Buscar paciente');

    fireEvent.focus(searchInput);

    expect(await screen.findByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('option', { name: /María Gómez/i }));

    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);

    fireEvent.focus(searchInput);

    fireEvent.change(searchInput, { target: { value: 'juan' } });

    expect(screen.getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /María Gómez/i })).not.toBeInTheDocument();
    expect(screen.getByText(/Tenés seleccionado a María Gómez; el filtro actual lo ocultó de la lista./i)).toBeInTheDocument();
    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);

    fireEvent.change(searchInput, { target: { value: 'zzzz' } });

    expect(screen.getByText(/No hay pacientes que coincidan con “zzzz”./i)).toBeInTheDocument();
    expect(screen.getByText(/Limpiá el buscador para ver la lista completa./i)).toBeInTheDocument();

    fireEvent.change(searchInput, { target: { value: '' } });

    expect(screen.getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();
  });

  it('renders the patient options inside the top search control instead of a separate results panel', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchControl = await screen.findByRole('group', { name: /buscador de pacientes/i });
    const searchInput = within(searchControl).getByRole('searchbox', { name: /buscar paciente/i });

    expect(within(searchControl).queryByRole('listbox', { name: /resultados de búsqueda de pacientes/i })).not.toBeInTheDocument();

    fireEvent.focus(searchInput);

    const resultsList = await within(searchControl).findByRole('listbox', { name: /resultados de búsqueda de pacientes/i });

    expect(searchInput).toBeInTheDocument();
    expect(resultsList).toBeInTheDocument();
    expect(searchInput).toHaveAttribute('aria-expanded', 'true');
    expect(screen.queryByText(/lista corta para entrar rápido a una atención/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/este cuerpo mantiene foco operativo/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /resultados/i })).not.toBeInTheDocument();
    expect(within(searchControl).getByText(/pacientes activos disponibles: 2/i)).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();

    fireEvent.click(within(resultsList).getByRole('option', { name: /María Gómez/i }));

    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);
    expect(within(searchControl).queryByRole('listbox', { name: /resultados de búsqueda de pacientes/i })).not.toBeInTheDocument();
    expect(searchInput).toHaveAttribute('aria-expanded', 'false');
  });

  it('falls back to another active patient when the selected one disappears after refresh', async () => {
    listPatientsMock.mockResolvedValueOnce({ items: [activePatient(), secondPatient()] }).mockResolvedValueOnce({
      items: [activePatient(), inactiveSecondPatient()],
    });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchInput = await screen.findByLabelText('Buscar paciente');

    fireEvent.focus(searchInput);
    fireEvent.click(await screen.findByRole('option', { name: /María Gómez/i }));

    expect(screen.getByText(/^Paciente$/i)).toBeInTheDocument();
    expect(screen.getByText('María Gómez')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /actualizar/i }));

    await waitFor(() => {
      expect(screen.getByText('Juan Pérez')).toBeInTheDocument();
    });

    expect(screen.queryByText('María Gómez')).not.toBeInTheDocument();
    expect(screen.getByText(/Pacientes activos: 1/i)).toBeInTheDocument();

    fireEvent.focus(searchInput);

    expect(await screen.findByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /María Gómez/i })).not.toBeInTheDocument();
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

function secondPatient() {
  return {
    id: 'patient-2',
    first_name: 'María',
    last_name: 'Gómez',
    document: '20999111',
    birth_date: '1991-02-02',
    phone: '555-9876',
    email: 'maria@example.com',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function inactiveSecondPatient() {
  return {
    ...secondPatient(),
    active: false,
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
