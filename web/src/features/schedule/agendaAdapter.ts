import type { Appointment, Consultation, Slot, WeekAgenda } from '../../types/appointments';
import { formatWeekdayLabel, formatTimeBand, getOperationalWeekDays } from './helpers';

export type ScheduleBoardAppointment = Appointment & {
	raw_status: Consultation['status'];
	status_label: string;
	is_standalone: boolean;
	can_cancel: boolean;
	source: Consultation['source'];
	scheduled_start: string;
	scheduled_end: string;
};

export type ScheduleBoardDay = {
	date: string;
	weekdayLabel: string;
	slots: Slot[];
	appointments: ScheduleBoardAppointment[];
	summary: {
		available: number;
		booked: number;
		cancelled: number;
		checkedIn: number;
		completed: number;
		noShow: number;
	};
};

export type ScheduleBoardModel = {
	weekStart: string;
	days: ScheduleBoardDay[];
	timeBands: string[];
};

export function buildScheduleBoardModel(weekAgenda: WeekAgenda): ScheduleBoardModel {
	const weekDays = getOperationalWeekDays(weekAgenda.week_start);
	const slotsByDate = groupSlotsByDate(weekAgenda.slots);
	const appointmentsByDate = groupAppointmentsByDate(weekAgenda.consultations);
	const timeBands = buildTimeBands(weekAgenda);

	const days = weekDays.map((day) => {
		const dayAppointments = [...(appointmentsByDate.get(day.date) ?? [])].sort(compareAppointmentsByStart);
		const standaloneSlots = dayAppointments
			.filter((appointment) => appointment.is_standalone)
			.map(createStandaloneSlot)
			.filter((slot): slot is Slot => slot !== null);
		const daySlots = [...(slotsByDate.get(day.date) ?? []), ...standaloneSlots].sort(compareSlotsByStartTime);

		return {
			date: day.date,
			weekdayLabel: day.weekdayLabel,
			slots: daySlots,
			appointments: dayAppointments,
			summary: buildDaySummary(daySlots, dayAppointments),
		};
	});

	return {
		weekStart: weekDays[0]?.date ?? weekAgenda.week_start,
		days,
		timeBands,
	};
}

function groupSlotsByDate(slots: Slot[]) {
	const byDate = new Map<string, Slot[]>();

	for (const slot of slots) {
		const date = slot.start_time.slice(0, 10);
		const current = byDate.get(date) ?? [];
		current.push(slot);
		byDate.set(date, current);
	}

	return byDate;
}

function groupAppointmentsByDate(consultations: Consultation[]) {
	const byDate = new Map<string, ScheduleBoardAppointment[]>();

	for (const consultation of consultations) {
		const date = consultation.scheduled_start.slice(0, 10);
		const current = byDate.get(date) ?? [];
		current.push(adaptConsultationToAppointment(consultation));
		byDate.set(date, current);
	}

	return byDate;
}

function adaptConsultationToAppointment(consultation: Consultation): ScheduleBoardAppointment {
	const isStandalone = !consultation.slot_id;
	const legacyStatus = consultation.status === 'cancelled' ? 'cancelled' : 'booked';

	return {
		id: consultation.id,
		slot_id: consultation.slot_id ?? standaloneSlotID(consultation.id),
		professional_id: consultation.professional_id,
		patient_id: consultation.patient_id,
		status: legacyStatus,
		created_at: consultation.created_at,
		updated_at: consultation.updated_at,
		cancelled_at: consultation.cancelled_at ?? null,
		raw_status: consultation.status,
		status_label: consultationStatusLabel(consultation.status),
		is_standalone: isStandalone,
		can_cancel: !isStandalone && consultation.status !== 'cancelled' && consultation.status !== 'completed' && consultation.status !== 'no_show',
		source: consultation.source,
		scheduled_start: consultation.scheduled_start,
		scheduled_end: consultation.scheduled_end,
	};
}

function createStandaloneSlot(appointment: ScheduleBoardAppointment): Slot | null {
	if (!appointment.is_standalone) {
		return null;
	}

	return {
		id: appointment.slot_id,
		professional_id: appointment.professional_id,
		start_time: appointment.scheduled_start,
		end_time: appointment.scheduled_end,
		status: 'booked',
		created_at: appointment.created_at,
		updated_at: appointment.updated_at,
	};
}

function buildTimeBands(weekAgenda: WeekAgenda) {
	const slotBands = weekAgenda.slots.map((slot) => formatTimeBand(slot.start_time));
	const consultationBands = weekAgenda.consultations.map((consultation) => formatTimeBand(consultation.scheduled_start));

	return [...new Set([...slotBands, ...consultationBands])].sort();
}

function buildDaySummary(slots: Slot[], appointments: ScheduleBoardAppointment[]) {
	return {
		available: slots.filter((slot) => slot.status === 'available').length,
		booked: appointments.filter((appointment) => appointment.raw_status === 'scheduled').length,
		cancelled: appointments.filter((appointment) => appointment.raw_status === 'cancelled').length,
		checkedIn: appointments.filter((appointment) => appointment.raw_status === 'checked_in').length,
		completed: appointments.filter((appointment) => appointment.raw_status === 'completed').length,
		noShow: appointments.filter((appointment) => appointment.raw_status === 'no_show').length,
	};
}

function consultationStatusLabel(status: Consultation['status']) {
	switch (status) {
		case 'scheduled':
			return 'Reservado';
		case 'checked_in':
			return 'En recepción';
		case 'completed':
			return 'Atendido';
		case 'cancelled':
			return 'Cancelado';
		case 'no_show':
			return 'Ausente';
	}
}

function standaloneSlotID(consultationID: string) {
	return `consultation-${consultationID}`;
}

function compareSlotsByStartTime(left: Slot, right: Slot) {
	return new Date(left.start_time).getTime() - new Date(right.start_time).getTime();
}

function compareAppointmentsByStart(left: ScheduleBoardAppointment, right: ScheduleBoardAppointment) {
	return new Date(left.scheduled_start).getTime() - new Date(right.scheduled_start).getTime();
}

export function buildBoardCellLabel(date: string, startTime: string, endTime: string) {
	return `${formatWeekdayLabel(date)} · ${startTime} - ${endTime} UTC`;
}
