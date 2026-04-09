import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { ScheduleDemo } from './ScheduleDemo';

const {
  cancelAppointmentMock,
  createAppointmentMock,
  listSlotsMock,
  listAppointmentsMock,
  listProfessionalsMock,
  listPatientsMock,
} = vi.hoisted(() => ({
  cancelAppointmentMock: vi.fn(),
  createAppointmentMock: vi.fn(),
  listSlotsMock: vi.fn(),
  listAppointmentsMock: vi.fn(),
  listProfessionalsMock: vi.fn(),
  listPatientsMock: vi.fn(),
}));

vi.mock('../../api/appointments', () => ({
  cancelAppointment: cancelAppointmentMock,
  createAppointment: createAppointmentMock,
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
    cancelAppointmentMock.mockReset();
    createAppointmentMock.mockReset();
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

  it('uses the slot grid as the only booking selector and reserves the clicked slot', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listSlotsMock.mockResolvedValue({
      items: [
        availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z'),
        availableSlot('slot-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z'),
      ],
    });
    listAppointmentsMock.mockResolvedValue({ items: [] });
    createAppointmentMock.mockResolvedValue({ id: 'appointment-2' });

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    const targetSlot = await screen.findByRole('button', { name: /10:00 - 10:30 UTC/i });
    expect(screen.queryByLabelText('Slot a reservar')).not.toBeInTheDocument();

    fireEvent.click(targetSlot);
    fireEvent.click(screen.getByRole('button', { name: 'Reservar turno' }));

    await waitFor(() => {
      expect(createAppointmentMock).toHaveBeenCalledWith({
        slot_id: 'slot-2',
        patient_id: 'patient-1',
        professional_id: 'professional-1',
      });
    });
  });

  it('keeps supporting guidance secondary without adding alternate agenda workflows', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listSlotsMock.mockResolvedValue({
      items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
    });
    listAppointmentsMock.mockResolvedValue({ items: [] });

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText('Paso 1: elegí profesional y fecha para cargar la disponibilidad del día.')).toBeInTheDocument();
    expect(screen.getByText('Paso 2 opcional: cargá disponibilidad para esa agenda sin salir de esta vista.')).toBeInTheDocument();
    expect(screen.getByText('Paso 3: elegí un horario desde la grilla principal y asignalo a un paciente.')).toBeInTheDocument();
    expect(screen.queryByLabelText('Slot a reservar')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Reprogramar/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Ver detalle/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('keeps cancellation refresh highlighting while hiding raw appointment and slot identifiers', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listSlotsMock
      .mockResolvedValueOnce({
        items: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
      })
      .mockResolvedValueOnce({
        items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
      });
    listAppointmentsMock
      .mockResolvedValueOnce({
        items: [bookedAppointment('appointment-1', 'slot-1')],
      })
      .mockResolvedValueOnce({
        items: [cancelledAppointment('appointment-1', 'slot-1')],
      });
    cancelAppointmentMock.mockResolvedValue(undefined);

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    fireEvent.click(await screen.findByRole('button', { name: 'Cancelar' }));

    expect(await screen.findByText(/volvió a quedar disponible/i)).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /09:00 - 09:30 UTC/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.queryByText(/Appointment:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Slot:/i)).not.toBeInTheDocument();
  });

  it('keeps the reserve and cancel workflow unchanged while surfacing user-facing appointment context', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listSlotsMock
      .mockResolvedValueOnce({
        items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
      })
      .mockResolvedValueOnce({
        items: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
      })
      .mockResolvedValueOnce({
        items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
      });
    listAppointmentsMock
      .mockResolvedValueOnce({ items: [] })
      .mockResolvedValueOnce({
        items: [bookedAppointment('appointment-1', 'slot-1')],
      })
      .mockResolvedValueOnce({
        items: [cancelledAppointment('appointment-1', 'slot-1')],
      });
    createAppointmentMock.mockResolvedValue({ id: 'appointment-1' });
    cancelAppointmentMock.mockResolvedValue(undefined);

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    fireEvent.click(await screen.findByRole('button', { name: /09:00 - 09:30 UTC/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Reservar turno' }));

    await waitFor(() => {
      expect(createAppointmentMock).toHaveBeenCalledWith({
        slot_id: 'slot-1',
        patient_id: 'patient-1',
        professional_id: 'professional-1',
      });
    });

    expect(await screen.findByText('Turno reservado correctamente.')).toBeInTheDocument();
    expect(await screen.findByText('Juan Pérez')).toBeInTheDocument();
    expect(screen.getByText('09:00 - 09:30 UTC')).toBeInTheDocument();
    expect(
      screen.getByText((content, element) => element?.tagName === 'SMALL' && content.includes('Ana Médica ·')),
    ).toBeInTheDocument();
    expect(screen.getByText('Reservado')).toBeInTheDocument();
    expect(screen.queryByText(/Appointment:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Slot:/i)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Cancelar' }));

    await waitFor(() => {
      expect(cancelAppointmentMock).toHaveBeenCalledWith('appointment-1');
    });

    expect(await screen.findByText(/volvió a quedar disponible/i)).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /09:00 - 09:30 UTC/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('Cancelado')).toBeInTheDocument();
  });
});

function availableSlot(id: string, startTime: string, endTime: string) {
  return {
    id,
    professional_id: 'professional-1',
    start_time: startTime,
    end_time: endTime,
    status: 'available',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function bookedSlot(id: string, startTime: string, endTime: string) {
  return {
    ...availableSlot(id, startTime, endTime),
    status: 'booked',
  };
}

function bookedAppointment(id: string, slotId: string) {
  return {
    id,
    slot_id: slotId,
    patient_id: 'patient-1',
    professional_id: 'professional-1',
    status: 'booked',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function cancelledAppointment(id: string, slotId: string) {
  return {
    ...bookedAppointment(id, slotId),
    status: 'cancelled',
  };
}

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
