import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { ScheduleDemo } from './ScheduleDemo';

const {
  listSlotsMock,
  listAppointmentsMock,
  listProfessionalsMock,
  listPatientsMock,
} = vi.hoisted(() => ({
  listSlotsMock: vi.fn(),
  listAppointmentsMock: vi.fn(),
  listProfessionalsMock: vi.fn(),
  listPatientsMock: vi.fn(),
}));

vi.mock('../../api/appointments', () => ({
  cancelAppointment: vi.fn(),
  createAppointment: vi.fn(),
  createSlotsBulk: vi.fn(),
  listAppointments: listAppointmentsMock,
  listSlots: listSlotsMock,
}));

vi.mock('../../api/directory', () => ({
  listPatients: listPatientsMock,
  listProfessionals: listProfessionalsMock,
}));

describe('ScheduleDemo auth failures', () => {
  beforeEach(() => {
    listSlotsMock.mockReset();
    listAppointmentsMock.mockReset();
    listProfessionalsMock.mockReset();
    listPatientsMock.mockReset();
  });

  it('invalidates the session on 401 schedule failures without showing denial feedback', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [activeProfessional()],
    });
    listPatientsMock.mockResolvedValue({
      items: [activePatient()],
    });
    listSlotsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    listAppointmentsMock.mockResolvedValue({ items: [] });

    const onSessionInvalid = vi.fn();

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });

    expect(screen.queryByText(/Acceso denegado:/i)).not.toBeInTheDocument();
  });

  it('shows denial feedback on 403 schedule failures without invalidating the session', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [activeProfessional()],
    });
    listPatientsMock.mockResolvedValue({
      items: [activePatient()],
    });
    listSlotsMock.mockRejectedValue(new ApiError('No podés ver esta agenda.', 403));
    listAppointmentsMock.mockResolvedValue({ items: [] });

    const onSessionInvalid = vi.fn();

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={onSessionInvalid} />);

    expect(await screen.findByText('Acceso denegado: No podés ver esta agenda.')).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
  });

  it('locks doctors to their own professional agenda', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [activeProfessional(), secondProfessional()],
    });
    listPatientsMock.mockResolvedValue({
      items: [activePatient()],
    });
    listSlotsMock.mockResolvedValue({ items: [] });
    listAppointmentsMock.mockResolvedValue({ items: [] });

    render(
      <ScheduleDemo agendaMode={{ kind: 'doctor-own', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Ana Médica').length).toBeGreaterThan(0);
    });
    expect(screen.queryByRole('combobox', { name: 'Profesional' })).not.toBeInTheDocument();
    await waitFor(() => {
      expect(listSlotsMock).toHaveBeenCalledWith({ professional_id: 'professional-1', date: expect.any(String) });
    });
  });
});

function secondProfessional() {
  return {
    id: 'professional-2',
    first_name: 'Beto',
    last_name: 'Trauma',
    specialty: 'Traumatología',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function activeProfessional() {
  return {
    id: 'professional-1',
    first_name: 'Ana',
    last_name: 'Médica',
    specialty: 'Clínica',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

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
