import { request } from './http';
import type {
  CreatePatientPayload,
  CreateProfessionalPayload,
  ListResponse,
  Patient,
  Professional,
} from '../types/directory';

const DIRECTORY_API_BASE = '/directory-api';

export function listProfessionals() {
	return request<ListResponse<Professional>>(DIRECTORY_API_BASE, '/professionals', {
		auth: true,
	});
}

export function listPublicProfessionals() {
	return request<ListResponse<Professional>>(DIRECTORY_API_BASE, '/public/professionals');
}

export function listPatients() {
	return request<ListResponse<Patient>>(DIRECTORY_API_BASE, '/patients', {
		auth: true,
	});
}

export function createPatient(payload: CreatePatientPayload) {
	return request<Patient>(DIRECTORY_API_BASE, '/patients', {
		method: 'POST',
		body: payload,
		auth: true,
	});
}

export function createProfessional(payload: CreateProfessionalPayload) {
	return request<Professional>(DIRECTORY_API_BASE, '/professionals', {
		method: 'POST',
		body: payload,
		auth: true,
	});
}
