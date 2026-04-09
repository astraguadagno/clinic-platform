import { request } from './http';
import type {
  Appointment,
  BulkCreateSlotsPayload,
  CreateAppointmentPayload,
  ListResponse,
  Slot,
} from '../types/appointments';
import { getOperationalWeekDays, normalizeOperationalTimeBands } from '../features/schedule/helpers';

const APPOINTMENTS_API_BASE = '/appointments-api';

type AppointmentFilters = {
  professional_id?: string;
  patient_id?: string;
  status?: string;
  date?: string;
};

type SlotFilters = {
  professional_id?: string;
  status?: string;
  date?: string;
};

export type OperationalWeekDay = {
  date: string;
  weekdayLabel: string;
  slots: Slot[];
  appointments: Appointment[];
  summary: {
    available: number;
    booked: number;
    cancelled: number;
  };
};

export type OperationalWeek = {
  weekStart: string;
  days: OperationalWeekDay[];
  timeBands: string[];
};

export function listSlots(filters: SlotFilters) {
	return request<ListResponse<Slot>>(APPOINTMENTS_API_BASE, '/slots', { query: filters, auth: true });
}

export function listAppointments(filters: AppointmentFilters) {
	return request<ListResponse<Appointment>>(APPOINTMENTS_API_BASE, '/appointments', { query: filters, auth: true });
}

export function createSlotsBulk(payload: BulkCreateSlotsPayload) {
	return request<ListResponse<Slot>>(APPOINTMENTS_API_BASE, '/slots/bulk', {
		method: 'POST',
		body: payload,
		auth: true,
	});
}

export function createAppointment(payload: CreateAppointmentPayload) {
	return request<Appointment>(APPOINTMENTS_API_BASE, '/appointments', {
		method: 'POST',
		body: payload,
		auth: true,
	});
}

export function cancelAppointment(appointmentId: string) {
	return request<Appointment>(APPOINTMENTS_API_BASE, `/appointments/${appointmentId}/cancel`, {
		method: 'PATCH',
		auth: true,
	});
}

export async function listOperationalWeek(filters: Required<Pick<SlotFilters, 'professional_id' | 'date'>>) {
	const weekDays = getOperationalWeekDays(filters.date);
	const days = await Promise.all(
		weekDays.map(async (day) => {
			const [slotsResponse, appointmentsResponse] = await Promise.all([
				listSlots({ professional_id: filters.professional_id, date: day.date }),
				listAppointments({ professional_id: filters.professional_id, date: day.date }),
			]);

			const slots = [...slotsResponse.items].sort(compareByStartTime);
			const appointments = [...appointmentsResponse.items].sort(compareBySlotOrder(slots));

			return {
				date: day.date,
				weekdayLabel: day.weekdayLabel,
				slots,
				appointments,
				summary: {
					available: slots.filter((slot) => slot.status === 'available').length,
					booked: appointments.filter((appointment) => appointment.status === 'booked').length,
					cancelled: appointments.filter((appointment) => appointment.status === 'cancelled').length,
				},
			};
		}),
	);

	return {
		weekStart: weekDays[0]?.date ?? filters.date,
		days,
		timeBands: normalizeOperationalTimeBands(days.flatMap((day) => day.slots)),
	};
}

function compareByStartTime(left: Slot, right: Slot) {
	return new Date(left.start_time).getTime() - new Date(right.start_time).getTime();
}

function compareBySlotOrder(slots: Slot[]) {
	const slotOrder = new Map(slots.map((slot, index) => [slot.id, index]));

	return (left: Appointment, right: Appointment) => {
		const leftOrder = slotOrder.get(left.slot_id) ?? Number.MAX_SAFE_INTEGER;
		const rightOrder = slotOrder.get(right.slot_id) ?? Number.MAX_SAFE_INTEGER;

		return leftOrder - rightOrder;
	};
}
