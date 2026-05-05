import { request } from './http';
import type {
  Appointment,
  BulkCreateSlotsPayload,
  Consultation,
  CreateScheduleTemplateVersionPayload,
  CreateAppointmentPayload,
  CreateConsultationPayload,
  CreatePatientRequestPayload,
  UpdateConsultationStatusPayload,
  GetScheduleTemplateFilters,
  ListResponse,
  ListScheduleTemplateVersionFilters,
  PublicAvailabilitySlot,
  ScheduleTemplate,
  ScheduleTemplateVersion,
  Slot,
  WeekAgenda,
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

type WeekAgendaFilters = {
	professional_id: string;
	week_start: string;
};

type PublicAvailabilityFilters = {
	professional_id: string;
	week_start: string;
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

export function createConsultation(payload: CreateConsultationPayload) {
	return request<Consultation>(APPOINTMENTS_API_BASE, '/consultations', {
		method: 'POST',
		body: payload,
		auth: true,
	});
}

export function createPatientRequest(payload: CreatePatientRequestPayload) {
	return request<Consultation>(APPOINTMENTS_API_BASE, '/patient-requests', {
		method: 'POST',
		body: payload,
	});
}

export function listPublicAvailability(filters: PublicAvailabilityFilters) {
	return request<ListResponse<PublicAvailabilitySlot>>(APPOINTMENTS_API_BASE, '/public/availability', { query: filters });
}

export function updateConsultationStatus(payload: UpdateConsultationStatusPayload) {
	return request<Consultation>(APPOINTMENTS_API_BASE, '/consultations', {
		method: 'PATCH',
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

export function fetchWeekAgenda(filters: WeekAgendaFilters) {
	return request<WeekAgenda>(APPOINTMENTS_API_BASE, '/agenda/week', { query: filters, auth: true });
}

export async function getScheduleTemplate(filters: GetScheduleTemplateFilters) {
	const template = await request<ScheduleTemplate>(APPOINTMENTS_API_BASE, '/schedules', { query: filters, auth: true });
	return normalizeScheduleTemplate(template);
}

export async function listScheduleTemplateVersions(filters: ListScheduleTemplateVersionFilters) {
	const response = await request<ListResponse<ScheduleTemplateVersion>>(APPOINTMENTS_API_BASE, '/schedules/versions', {
		query: filters,
		auth: true,
	});

	return {
		items: response.items.map(normalizeScheduleTemplateVersion),
	};
}

export async function createScheduleTemplateVersion(payload: CreateScheduleTemplateVersionPayload) {
	const template = await request<ScheduleTemplate>(APPOINTMENTS_API_BASE, '/schedules', {
		method: 'POST',
		body: payload,
		auth: true,
	});

	return normalizeScheduleTemplate(template);
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

function normalizeScheduleTemplate(template: ScheduleTemplate): ScheduleTemplate {
	return {
		...template,
		versions: template.versions?.map(normalizeScheduleTemplateVersion),
	};
}

function normalizeScheduleTemplateVersion(version: ScheduleTemplateVersion): ScheduleTemplateVersion {
	return {
		...version,
		effective_from: normalizeDateOnly(version.effective_from),
	};
}

function normalizeDateOnly(value: string) {
	return value.includes('T') ? value.slice(0, 10) : value;
}
