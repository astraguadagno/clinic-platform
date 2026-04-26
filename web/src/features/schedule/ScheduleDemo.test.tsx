import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import type { Consultation, Slot, WeekAgenda } from '../../types/appointments';
import { ScheduleDemo } from './ScheduleDemo';

const {
	cancelAppointmentMock,
	createAppointmentMock,
	createSlotsBulkMock,
	fetchWeekAgendaMock,
	listProfessionalsMock,
	listPatientsMock,
} = vi.hoisted(() => ({
	cancelAppointmentMock: vi.fn(),
	createAppointmentMock: vi.fn(),
	createSlotsBulkMock: vi.fn(),
	fetchWeekAgendaMock: vi.fn(),
	listProfessionalsMock: vi.fn(),
	listPatientsMock: vi.fn(),
}));

vi.mock('../../api/appointments', () => ({
	cancelAppointment: cancelAppointmentMock,
	createAppointment: createAppointmentMock,
	createSlotsBulk: createSlotsBulkMock,
	fetchWeekAgenda: fetchWeekAgendaMock,
}));

vi.mock('../../api/directory', () => ({
	listPatients: listPatientsMock,
	listProfessionals: listProfessionalsMock,
}));

vi.mock('./helpers', async () => {
	const actual = await vi.importActual<typeof import('./helpers')>('./helpers');

	return {
		...actual,
		formatDateInputValue: (date?: Date) => actual.formatDateInputValue(date ?? new Date('2026-04-08T12:00:00Z')),
	};
});

describe('ScheduleDemo weekly operational board', () => {
	beforeEach(() => {
		cancelAppointmentMock.mockReset();
		createAppointmentMock.mockReset();
		createSlotsBulkMock.mockReset();
		fetchWeekAgendaMock.mockReset();
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
		expect(screen.getByText('Acceso denegado')).toBeInTheDocument();
		expect(screen.getByText('Tu perfil no tiene permisos para usar esta agenda.')).toBeInTheDocument();
		expect(fetchWeekAgendaMock).not.toHaveBeenCalled();
		expect(listProfessionalsMock).not.toHaveBeenCalled();
		expect(listPatientsMock).not.toHaveBeenCalled();
	});

	it('invalidates the session on 401 weekly load failures without showing denial feedback', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
		listPatientsMock.mockResolvedValue({ items: [activePatient()] });
		fetchWeekAgendaMock.mockRejectedValue(new ApiError('Sesión vencida', 401));

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
		fetchWeekAgendaMock.mockResolvedValue(weekAgenda({}));

		render(
			<ScheduleDemo agendaMode={{ kind: 'doctor-own', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />,
		);

		expect((await screen.findAllByText('Lun 06/04')).length).toBeGreaterThan(0);
		expect(screen.queryByRole('combobox', { name: 'Profesional' })).not.toBeInTheDocument();
		expect(screen.getByText(/agenda clínica propia/i)).toBeInTheDocument();
		expect(fetchWeekAgendaMock).toHaveBeenCalledWith({ professional_id: 'professional-1', week_start: '2026-04-06' });
	});

	it('offers secretary a hidden directory support entry when setup data is missing', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [] });
		listPatientsMock.mockResolvedValue({ items: [] });
		fetchWeekAgendaMock.mockResolvedValue(weekAgenda({}));

		const onOpenDirectorySupport = vi.fn();

		render(
			<ScheduleDemo
				agendaMode={{ kind: 'operational-shared' }}
				onSessionInvalid={vi.fn()}
				onOpenDirectorySupport={onOpenDirectorySupport}
			/>,
		);

		expect(await screen.findByText(/faltan profesionales y pacientes para operar la agenda/i)).toBeInTheDocument();

		fireEvent.click(screen.getByRole('button', { name: /abrir soporte de directorio/i }));

		expect(onOpenDirectorySupport).toHaveBeenCalledTimes(1);
	});

	it('keeps the agenda board focused on daily operations without embedding weekly template editing', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [activeProfessional(), secondProfessional()] });
		listPatientsMock.mockResolvedValue({ items: [activePatient()] });
		fetchWeekAgendaMock.mockResolvedValue(weekAgenda({}));

		render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

		expect(await screen.findByRole('heading', { name: 'Agenda semanal operativa' })).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Editar agenda semanal' })).not.toBeInTheDocument();
	});

	it('keeps doctors on the daily board without a duplicated weekly editor entry', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [activeProfessional(), secondProfessional()] });
		listPatientsMock.mockResolvedValue({ items: [activePatient()] });
		fetchWeekAgendaMock.mockResolvedValue(weekAgenda({}));

			render(
			<ScheduleDemo agendaMode={{ kind: 'doctor-own', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />,
		);

		expect(await screen.findByText(/agenda clínica propia/i)).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Editar agenda semanal' })).not.toBeInTheDocument();
	});

	it('opens booking from an available slot and refreshes the weekly board', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
		listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });
		fetchWeekAgendaMock
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-08': { slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z'), availableSlot('slot-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z')] },
			}))
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-08': {
					slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z'), bookedSlot('slot-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z')],
					consultations: [scheduledConsultation('appointment-1', 'slot-2', 'patient-2', '2026-04-08T10:00:00Z', '2026-04-08T10:30:00Z')],
				},
			}));
		createAppointmentMock.mockResolvedValue({ id: 'appointment-1' });

		render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

		fireEvent.click((await screen.findAllByRole('button', { name: /Mié 08\/04/i }))[0]);
		const slotCell = await screen.findByRole('button', { name: /Mié 08\/04 .*10:00 - 10:30 UTC/i });
		fireEvent.click(slotCell);
		fireEvent.click(screen.getByRole('button', { name: 'Reservar turno' }));

		const bookingDialog = await screen.findByRole('dialog', { name: /Reservar turno/i });
		fireEvent.change(within(bookingDialog).getByLabelText('Buscar paciente'), { target: { value: 'maria' } });
		fireEvent.click(within(bookingDialog).getByRole('button', { name: /María Gómez/i }));
		fireEvent.click(within(bookingDialog).getByRole('button', { name: 'Confirmar reserva' }));

		await waitFor(() => {
			expect(createAppointmentMock).toHaveBeenCalledWith({
				slot_id: 'slot-2',
				patient_id: 'patient-2',
				professional_id: 'professional-1',
			});
		});

		expect(await screen.findByText('Turno reservado correctamente.')).toBeInTheDocument();
		expect(await screen.findByText('María Gómez')).toBeInTheDocument();
		expect(screen.getByText('Reservado')).toBeInTheDocument();
		expect(fetchWeekAgendaMock).toHaveBeenCalledTimes(2);
	});

	it('opens slot generation through a modal bound to the selected day', async () => {
		listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });
		listPatientsMock.mockResolvedValue({ items: [activePatient()] });
		fetchWeekAgendaMock
			.mockResolvedValueOnce(weekAgenda({}))
			.mockResolvedValueOnce(weekAgenda({ '2026-04-08': { slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')] } }));
		createSlotsBulkMock.mockResolvedValue({ items: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')] });

		render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

		fireEvent.click((await screen.findAllByRole('button', { name: /Mié 08\/04/i }))[0]);
		fireEvent.click(screen.getByRole('button', { name: 'Generar slots' }));

		const generateDialog = await screen.findByRole('dialog', { name: /Generar slots/i });
		fireEvent.click(within(generateDialog).getByRole('button', { name: 'Confirmar generación' }));

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
		fetchWeekAgendaMock
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-08': {
					slots: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
					consultations: [scheduledConsultation('appointment-1', 'slot-1', 'patient-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
				},
			}))
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-13': { slots: [availableSlot('slot-next', '2026-04-13T09:00:00Z', '2026-04-13T09:30:00Z')] },
			}, '2026-04-13'))
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-06': {
					slots: [bookedSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
					consultations: [scheduledConsultation('appointment-1', 'slot-1', 'patient-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
				},
			}))
			.mockResolvedValueOnce(weekAgenda({
				'2026-04-08': {
					slots: [availableSlot('slot-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
					consultations: [cancelledConsultation('appointment-1', 'slot-1', 'patient-1', '2026-04-08T09:00:00Z', '2026-04-08T09:30:00Z')],
				},
			}));
		cancelAppointmentMock.mockResolvedValue(undefined);

		render(<ScheduleDemo agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

		expect(await screen.findByText('Juan Pérez')).toBeInTheDocument();
		fireEvent.click(screen.getAllByRole('button', { name: /Mié 08\/04/i })[0]);
		fireEvent.click(screen.getByRole('button', { name: 'Semana siguiente' }));
		expect(await screen.findByText('Lun 13/04')).toBeInTheDocument();
		fireEvent.click(screen.getByRole('button', { name: 'Semana anterior' }));
		fireEvent.click(await screen.findByRole('button', { name: 'Cancelar' }));

		await waitFor(() => {
			expect(cancelAppointmentMock).toHaveBeenCalledWith('appointment-1');
		});

		expect(await screen.findByText(/volvió a quedar disponible/i)).toBeInTheDocument();
		expect(within(screen.getByTestId('board-cell-2026-04-08-09:00')).getByText('Cancelado')).toBeInTheDocument();
	});
});

function weekAgenda(
	days: Record<string, { slots?: Slot[]; consultations?: Consultation[] }>,
	weekStart = '2026-04-06',
): WeekAgenda {
	return {
		professional_id: 'professional-1',
		week_start: weekStart,
		templates: [],
		blocks: [],
		consultations: Object.values(days).flatMap((day) => day.consultations ?? []),
		slots: Object.values(days).flatMap((day) => day.slots ?? []),
	};
}

function availableSlot(id: string, startTime: string, endTime: string): Slot {
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

function bookedSlot(id: string, startTime: string, endTime: string): Slot {
	return {
		...availableSlot(id, startTime, endTime),
		status: 'booked',
	};
}

function scheduledConsultation(id: string, slotId: string | null, patientId: string, startTime: string, endTime: string): Consultation {
	return {
		id,
		slot_id: slotId,
		professional_id: 'professional-1',
		patient_id: patientId,
		status: 'scheduled',
		source: 'secretary',
		scheduled_start: startTime,
		scheduled_end: endTime,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
	};
}

function cancelledConsultation(id: string, slotId: string | null, patientId: string, startTime: string, endTime: string): Consultation {
	return {
		...scheduledConsultation(id, slotId, patientId, startTime, endTime),
		status: 'cancelled',
		cancelled_at: '2026-01-02T00:00:00Z',
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
