import { request } from './http';
import type {
  Appointment,
  BulkCreateSlotsPayload,
  CreateAppointmentPayload,
  ListResponse,
  Slot,
} from '../types/appointments';

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

export function listSlots(filters: SlotFilters) {
  return request<ListResponse<Slot>>(APPOINTMENTS_API_BASE, '/slots', { query: filters });
}

export function listAppointments(filters: AppointmentFilters) {
  return request<ListResponse<Appointment>>(APPOINTMENTS_API_BASE, '/appointments', { query: filters });
}

export function createSlotsBulk(payload: BulkCreateSlotsPayload) {
  return request<ListResponse<Slot>>(APPOINTMENTS_API_BASE, '/slots/bulk', {
    method: 'POST',
    body: payload,
  });
}

export function createAppointment(payload: CreateAppointmentPayload) {
  return request<Appointment>(APPOINTMENTS_API_BASE, '/appointments', {
    method: 'POST',
    body: payload,
  });
}

export function cancelAppointment(appointmentId: string) {
  return request<Appointment>(APPOINTMENTS_API_BASE, `/appointments/${appointmentId}/cancel`, {
    method: 'PATCH',
  });
}
