import { describe, expect, it } from 'vitest';
import type { Consultation, Slot, WeekAgenda } from '../../types/appointments';
import { buildScheduleBoardModel } from './agendaAdapter';

describe('agendaAdapter', () => {
	it('builds a board model from slots and consultations', () => {
		const weekAgenda: WeekAgenda = {
			professional_id: 'professional-1',
			week_start: '2026-04-06',
			templates: [],
			blocks: [],
			consultations: [
				consultation({
					id: 'consultation-1',
					slot_id: 'slot-1',
					status: 'scheduled',
					scheduled_start: '2026-04-08T10:00:00Z',
					scheduled_end: '2026-04-08T10:30:00Z',
				}),
			],
			slots: [slot({ id: 'slot-1', start_time: '2026-04-08T10:00:00Z', end_time: '2026-04-08T10:30:00Z', status: 'booked' })],
		};

		const board = buildScheduleBoardModel(weekAgenda);

		expect(board.weekStart).toBe('2026-04-06');
		expect(board.timeBands).toEqual(['10:00']);
		expect(board.days).toHaveLength(5);
		expect(board.days[2]).toMatchObject({
			date: '2026-04-08',
			summary: { available: 0, booked: 1, cancelled: 0, checkedIn: 0, completed: 0, noShow: 0 },
		});
		expect(board.days[2]?.appointments[0]).toMatchObject({
			id: 'consultation-1',
			slot_id: 'slot-1',
			status: 'booked',
			raw_status: 'scheduled',
			status_label: 'Reservado',
			is_standalone: false,
			can_cancel: true,
		});
	});

	it('creates synthetic booked slots for standalone consultations and exposes their time bands', () => {
		const weekAgenda: WeekAgenda = {
			professional_id: 'professional-1',
			week_start: '2026-04-06',
			templates: [],
			blocks: [],
			consultations: [
				consultation({
					id: 'consultation-standalone',
					slot_id: null,
					status: 'checked_in',
					scheduled_start: '2026-04-07T09:15:00Z',
					scheduled_end: '2026-04-07T09:35:00Z',
				}),
			],
			slots: [],
		};

		const board = buildScheduleBoardModel(weekAgenda);

		expect(board.timeBands).toEqual(['09:15']);
		expect(board.days[1]?.slots[0]).toMatchObject({
			id: 'consultation-consultation-standalone',
			status: 'booked',
			start_time: '2026-04-07T09:15:00Z',
			end_time: '2026-04-07T09:35:00Z',
		});
		expect(board.days[1]?.appointments[0]).toMatchObject({
			id: 'consultation-standalone',
			is_standalone: true,
			can_cancel: false,
			raw_status: 'checked_in',
			status_label: 'En recepción',
		});
		expect(board.days[1]?.summary.checkedIn).toBe(1);
	});

	it('keeps cancelled consultations visible in the summary', () => {
		const weekAgenda: WeekAgenda = {
			professional_id: 'professional-1',
			week_start: '2026-04-06',
			templates: [],
			blocks: [],
			consultations: [
				consultation({
					id: 'consultation-2',
					slot_id: 'slot-2',
					status: 'cancelled',
					scheduled_start: '2026-04-10T11:00:00Z',
					scheduled_end: '2026-04-10T11:30:00Z',
				}),
			],
			slots: [slot({ id: 'slot-2', start_time: '2026-04-10T11:00:00Z', end_time: '2026-04-10T11:30:00Z', status: 'available' })],
		};

		const board = buildScheduleBoardModel(weekAgenda);

		expect(board.timeBands).toEqual(['11:00']);
		expect(board.days[4]?.summary).toMatchObject({ cancelled: 1, available: 1 });
		expect(board.days[4]?.appointments[0]).toMatchObject({
			status: 'cancelled',
			raw_status: 'cancelled',
			status_label: 'Cancelado',
			can_cancel: false,
		});
	});
});

function slot(overrides: Partial<Slot>): Slot {
	return {
		id: 'slot-default',
		professional_id: 'professional-1',
		start_time: '2026-04-08T10:00:00Z',
		end_time: '2026-04-08T10:30:00Z',
		status: 'available',
		created_at: '2026-04-01T08:00:00Z',
		updated_at: '2026-04-01T08:00:00Z',
		...overrides,
	};
}

function consultation(overrides: Partial<Consultation>): Consultation {
	return {
		id: 'consultation-default',
		slot_id: 'slot-default',
		professional_id: 'professional-1',
		patient_id: 'patient-1',
		status: 'scheduled',
		source: 'secretary',
		scheduled_start: '2026-04-08T10:00:00Z',
		scheduled_end: '2026-04-08T10:30:00Z',
		created_at: '2026-04-01T08:00:00Z',
		updated_at: '2026-04-01T08:00:00Z',
		...overrides,
	};
}
