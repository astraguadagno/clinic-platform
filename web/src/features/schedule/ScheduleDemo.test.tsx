import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import type { Appointment, Slot } from '../../types/appointments';
import { ScheduleDemo } from './ScheduleDemo';

const {
  cancelAppointmentMock,
  createAppointmentMock,
  createSlotsBulkMock,
  listSlotsMock,
  listAppointmentsMock,
  listProfessionalsMock,
  listPatientsMock,
} = vi.hoisted(() => ({
  cancelAppointmentMock: vi.fn(),
  createAppointmentMock: vi.fn(),
  createSlotsBulkMock: vi.fn(),
  listSlotsMock: vi.fn(),
  listAppointmentsMock: vi.fn(),
  listProfessionalsMock: vi.fn(),
  listPatientsMock: vi.fn(),
}));

vi.mock('../../api/appointments', () => ({
  cancelAppointment: cancelAppointmentMock,
  createAppointment: createAppointmentMock,
  createSlotsBulk: createSlotsBulkMock,
  listAppointments: listAppointmentsMock,
  listSlots: listSlotsMock,
  listOperationalWeek: async ({ professional_id, date }: { professional_id: string; date: string }) => {
    const weekDays = getWeekDays(date);
    const days = await Promise.all(
      weekDays.map(async (day) => {
        const [slotsResponse, appointmentsResponse] = await Promise.all([
          listSlotsMock({ professional_id, date: day.date }),
          listAppointmentsMock({ professional_id, date: day.date }),
        ]);

        return {
          date: day.date,
          weekdayLabel: day.weekdayLabel,
          slots: slotsResponse.items,
          appointments: appointmentsResponse.items,
          summary: {
            available: slotsResponse.items.filter((slot: Slot) => slot.status === 'available').length,
            booked: appointmentsResponse.items.filter((appointment: { status: string }) => appointment.status === 'booked').length,
            cancelled: appointmentsResponse.items.filter((appointment: { status: string }) => appointment.status === 'cancelled').length,
          },
        };
      }),
    );

    return {
      weekStart: weekDays[0]?.date ?? date,
      days,
      timeBands: [...new Set(days.flatMap((day) => day.slots.map((slot: Slot) => slot.start_time.slice(11, 16))))].sort(),
    };
  },
}));

vi.mock('../../api/directory', () => ({
  listPatients: listPatientsMock,
  listProfessionals: listProfessionalsMock,
}));

describe('ScheduleDemo weekly operational board', () => {
  beforeEach(() => {
    cancelAppointmentMock.mockReset();
    createAppointmentMock.mockReset();
    createSlotsBulkMock.mockReset();
    listSlotsMock.mockReset();
    listAppointmentsMock.mockReset();
    listProfessionalsMock.mockReset();
    listPatientsMock.mockReset();
  });

  it('renders the explicit forbidden agenda branch without bootstrapping schedule data', () => {
    render(
      <ScheduleDemo
        agendaMode={{ kind: 'forbidden', message: 'Tu perfil no tiene permisos para usar esta agenda.' }}
        onSessionInvalid={vi.fn()}
      />,
    );

    expect(screen.getByText('Agenda bloqueada')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Acceso denegado' })).toBeInTheDocument();
    expect(screen.getByText('Tu perfil no tiene permisos para usar esta agenda.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Actualizar/i })).not.toBeInTheDocument();
    expect(listProfessionalsMock).not.toHaveBeenCalled();
    expect(listPatientsMock).not.toHaveBeenCalled();
    expect(listSlotsMock).not.toHaveBeenCalled();
    expect(listAppointmentsMock).not.toHaveBeenCalled();
  });

  it('invalidates the session on 401 weekly load failures without showing denial feedback', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listSlotsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    listAppointmentsMock.mockResolvedValue({ items: [] });

    const onSessionInvalid = vi.fn();

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });

    expect(screen.queryByText(/Acceso denegado:/i)).not.toBeInTheDocument();
  });

  it('locks doctors to their own professional agenda while loading the visible monday-friday week', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional(), secondProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    mockWeekSlices({});

    render(
      <ScheduleDemo agendaMode={{ kind: 'doctor-own', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />,
    );

    expect((await screen.findAllByText('Lun 06/04')).length).toBeGreaterThan(0);

    expect(screen.queryByRole('combobox', { name: 'Profesional' })).not.toBeInTheDocument();
    expect(listSlotsMock).toHaveBeenCalledTimes(5);
    expect(listSlotsMock.mock.calls.map(([filters]) => filters)).toEqual([
      { professional_id: 'professional-1', date: '2026-04-06' },
      { professional_id: 'professional-1', date: '2026-04-07' },
      { professional_id: 'professional-1', date: '2026-04-08' },
      { professional_id: 'professional-1', date: '2026-04-09' },
      { professional_id: 'professional-1', date: '2026-04-10' },
    ]);
  });

  it('shows a monday-friday weekly board with explicit empty weekdays', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    mockWeekSlices({
      '2026-04-06': { slots: [availableSlot('slot-mon', '2026-04-06T09:00:00Z', '2026-04-06T09:30:00Z')] },
      '2026-04-08': { slots: [availableSlot('slot-wed', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z')] },
    });

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByRole('heading', { name: 'Agenda semanal operativa' })).toBeInTheDocument();
    expect((await screen.findAllByText('Lun 06/04')).length).toBeGreaterThan(0);
    expect(screen.getAllByText('Mar 07/04').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Mié 08/04').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Jue 09/04').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Vie 10/04').length).toBeGreaterThan(0);
    expect(screen.queryByText(/Sáb/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Dom/i)).not.toBeInTheDocument();
    expect(screen.getAllByText('Sin actividad para este día').length).toBeGreaterThan(0);
  });

  it('uses the selected slot cell as the only booking selector and preserves supported actions only', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    mockWeekRefreshes([
      {
        '2026-04-08': {
          slots: [
            availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z'),
            availableSlot('slot-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z'),
          ],
        },
      },
      {
        '2026-04-08': {
          slots: [bookedSlot('slot-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z')],
          appointments: [bookedAppointment('appointment-1', 'slot-2')],
        },
      },
    ]);
    createAppointmentMock.mockResolvedValue({ id: 'appointment-1' });

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    fireEvent.click((await screen.findAllByRole('button', { name: /Mié 08\/04/i }))[0]);

    const slotCell = await screen.findByRole('button', { name: /Mié 08\/04 .*10:00 - 10:30 UTC/i });

    expect(screen.queryByLabelText('Slot a reservar')).not.toBeInTheDocument();
    expect(screen.getByText('Elegí profesional y semana para cargar la operación visible sin salir del espacio actual.')).toBeInTheDocument();
    expect(screen.getByText('Acción secundaria: cargá disponibilidad para el día seleccionado sin salir de esta vista.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Reprogramar/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Ver detalle/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();

    fireEvent.click(slotCell);
    fireEvent.click(screen.getByRole('button', { name: 'Reservar turno' }));

    await waitFor(() => {
      expect(createAppointmentMock).toHaveBeenCalledWith({
        slot_id: 'slot-2',
        patient_id: 'patient-1',
        professional_id: 'professional-1',
      });
    });

    expect(await screen.findByText('Turno reservado correctamente.')).toBeInTheDocument();
    expect(await screen.findByText('Juan Pérez')).toBeInTheDocument();
    expect(screen.getByText('Reservado')).toBeInTheDocument();
  });

  it('lets the operator select a weekday card and keeps support actions bound to that selected day', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    mockWeekRefreshes([
      {},
      { '2026-04-08': { slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')] } },
    ]);
    createSlotsBulkMock.mockResolvedValue({
      items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
    });

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    const weekdayCard = (await screen.findAllByRole('button', { name: /Mié 08\/04/i }))[0];
    fireEvent.click(weekdayCard);

    expect(weekdayCard).toHaveAttribute('aria-pressed', 'true');
    expect(await screen.findAllByText('Sin actividad para este día')).not.toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Generar slots' }));

    await waitFor(() => {
      expect(createSlotsBulkMock).toHaveBeenCalledWith({
        professional_id: 'professional-1',
        date: '2026-04-08',
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      });
    });

    expect(await screen.findByText('Se generaron 1 slot correctamente.')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /Mié 08\/04 .*09:00 - 09:30 UTC/i })).toBeInTheDocument();
  });

  it('supports week browsing and cancellation while keeping released slot feedback intact', async () => {
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    mockWeekRefreshes([
      { '2026-04-08': { slots: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')], appointments: [bookedAppointment('appointment-1', 'slot-1')] } },
      { '2026-04-13': { slots: [availableSlot('slot-next', '2026-04-13T09:00:00Z', '2026-04-13T09:30:00Z')] } },
      { '2026-04-08': { slots: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')], appointments: [bookedAppointment('appointment-1', 'slot-1')] } },
      { '2026-04-08': { slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')], appointments: [cancelledAppointment('appointment-1', 'slot-1')] } },
    ]);
    cancelAppointmentMock.mockResolvedValue(undefined);

    render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    const appointmentCell = await screen.findByTestId('board-cell-2026-04-08-09:00');
    expect(within(appointmentCell).getByText('Juan Pérez')).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole('button', { name: /Mié 08\/04/i })[0]);

    fireEvent.click(screen.getByRole('button', { name: 'Semana siguiente' }));
    expect(await screen.findByText('Lun 13/04')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Mié 15\/04/i })).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(screen.getByRole('button', { name: 'Semana anterior' }));
    expect(await screen.findByRole('button', { name: /Mié 08\/04/i })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(await screen.findByRole('button', { name: 'Cancelar' }));

    await waitFor(() => {
      expect(cancelAppointmentMock).toHaveBeenCalledWith('appointment-1');
    });

    expect(await screen.findByText(/volvió a quedar disponible/i)).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /Mié 08\/04 .*09:00 - 09:30 UTC/i })).toHaveAttribute('aria-pressed', 'true');
    expect(within(screen.getByTestId('board-cell-2026-04-08-09:00')).getByText('Cancelado')).toBeInTheDocument();
    expect(screen.queryByText(/Appointment:/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Slot:/i)).not.toBeInTheDocument();
  });
});

function mockWeekSlices(days: Record<string, { slots?: Slot[]; appointments?: Appointment[] }>) {
  listSlotsMock.mockImplementation(({ date }: { date: string }) => Promise.resolve({ items: days[date]?.slots ?? [] }));
  listAppointmentsMock.mockImplementation(({ date }: { date: string }) => Promise.resolve({ items: days[date]?.appointments ?? [] }));
}

function mockWeekRefreshes(weeks: Array<Record<string, { slots?: Slot[]; appointments?: Appointment[] }>>) {
  let activeWeekIndex = 0;
  let callsInWeek = 0;

  const advanceWeekIfNeeded = () => {
    callsInWeek += 1;
    if (callsInWeek === 10) {
      callsInWeek = 0;
      if (activeWeekIndex < weeks.length - 1) {
        activeWeekIndex += 1;
      }
    }
  };

  listSlotsMock.mockImplementation(({ date }: { date: string }) => {
    const result = Promise.resolve({ items: weeks[activeWeekIndex]?.[date]?.slots ?? [] });
    advanceWeekIfNeeded();
    return result;
  });

  listAppointmentsMock.mockImplementation(({ date }: { date: string }) => {
    const result = Promise.resolve({ items: weeks[activeWeekIndex]?.[date]?.appointments ?? [] });
    advanceWeekIfNeeded();
    return result;
  });
}

function availableSlot(id: string, startTime: string, endTime: string) {
  return {
    id,
    professional_id: 'professional-1',
    start_time: startTime,
    end_time: endTime,
    status: 'available' as const,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function bookedSlot(id: string, startTime: string, endTime: string) {
  return {
    ...availableSlot(id, startTime, endTime),
    status: 'booked' as const,
  };
}

function bookedAppointment(id: string, slotId: string) {
  return {
    id,
    slot_id: slotId,
    patient_id: 'patient-1',
    professional_id: 'professional-1',
    status: 'booked' as const,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function cancelledAppointment(id: string, slotId: string) {
  return {
    ...bookedAppointment(id, slotId),
    status: 'cancelled' as const,
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

function getWeekDays(date: string) {
  const parsedDate = new Date(`${date}T00:00:00Z`);
  const dayOfWeek = parsedDate.getUTCDay();
  const offsetToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
  parsedDate.setUTCDate(parsedDate.getUTCDate() + offsetToMonday);

  return Array.from({ length: 5 }, (_, index) => {
    const currentDate = new Date(parsedDate);
    currentDate.setUTCDate(parsedDate.getUTCDate() + index);

    return {
      date: currentDate.toISOString().slice(0, 10),
      weekdayLabel: ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'][currentDate.getUTCDay()] + ` ${currentDate.toISOString().slice(8, 10)}/${currentDate.toISOString().slice(5, 7)}`,
    };
  });
}
