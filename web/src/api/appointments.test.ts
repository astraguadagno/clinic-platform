import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { WeekAgenda } from '../types/appointments';

const requestMock = vi.fn();

vi.mock('./http', () => ({
  request: requestMock,
}));

describe('appointments API auth transport', () => {
  beforeEach(() => {
    requestMock.mockReset();
  });

  it('opts into authenticated transport for schedule reads and writes', async () => {
    requestMock.mockResolvedValue({ items: [] });

    const appointments = await import('./appointments');

    await appointments.listSlots({ professional_id: 'professional-1', date: '2026-04-08' });
    await appointments.createAppointment({
      slot_id: 'slot-1',
      patient_id: 'patient-1',
      professional_id: 'professional-1',
    });
    await appointments.getScheduleTemplate({ professional_id: 'professional-1', effective_date: '2099-12-31' });
    await appointments.listScheduleTemplateVersions({ template_id: 'template-1' });
    await appointments.createScheduleTemplateVersion({
      professional_id: 'professional-1',
      effective_from: '2026-05-01',
      recurrence: {
        monday: {
          start_time: '09:00',
          end_time: '12:00',
          slot_duration_minutes: 30,
        },
      },
      reason: 'Nueva versión semanal',
    });

    expect(requestMock).toHaveBeenNthCalledWith(1, '/appointments-api', '/slots', {
      query: { professional_id: 'professional-1', date: '2026-04-08' },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(2, '/appointments-api', '/appointments', {
      method: 'POST',
      body: {
        slot_id: 'slot-1',
        patient_id: 'patient-1',
        professional_id: 'professional-1',
      },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(3, '/appointments-api', '/schedules', {
      query: { professional_id: 'professional-1', effective_date: '2099-12-31' },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(4, '/appointments-api', '/schedules/versions', {
      query: { template_id: 'template-1' },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(5, '/appointments-api', '/schedules', {
      method: 'POST',
      body: {
        professional_id: 'professional-1',
        effective_from: '2026-05-01',
        recurrence: {
          monday: {
            start_time: '09:00',
            end_time: '12:00',
            slot_duration_minutes: 30,
          },
        },
        reason: 'Nueva versión semanal',
      },
      auth: true,
    });
  });

  it('composes a monday-friday operational week from the current day endpoints', async () => {
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [{ id: 'slot-1', professional_id: 'professional-1', start_time: '2026-04-07T09:00:00Z', end_time: '2026-04-07T09:30:00Z', status: 'available', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [{ id: 'slot-2', professional_id: 'professional-1', start_time: '2026-04-10T11:00:00Z', end_time: '2026-04-10T11:30:00Z', status: 'booked', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }] });
    requestMock.mockResolvedValueOnce({ items: [{ id: 'appointment-1', slot_id: 'slot-2', professional_id: 'professional-1', patient_id: 'patient-1', status: 'booked', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }] });

    const appointments = await import('./appointments');

    const week = await appointments.listOperationalWeek({ professional_id: 'professional-1', date: '2026-04-08' });

    expect(requestMock).toHaveBeenCalledTimes(10);
    expect(requestMock).toHaveBeenNthCalledWith(1, '/appointments-api', '/slots', {
      query: { professional_id: 'professional-1', date: '2026-04-06' },
      auth: true,
    });
    expect(requestMock).toHaveBeenNthCalledWith(10, '/appointments-api', '/appointments', {
      query: { professional_id: 'professional-1', date: '2026-04-10' },
      auth: true,
    });
    expect(week.weekStart).toBe('2026-04-06');
    expect(week.days.map((day) => day.date)).toEqual(['2026-04-06', '2026-04-07', '2026-04-08', '2026-04-09', '2026-04-10']);
    expect(week.timeBands).toEqual(['09:00', '11:00']);
    expect(week.days[1]?.summary.available).toBe(1);
    expect(week.days[4]?.summary.booked).toBe(1);
  });

  it('fails fast when a day slice cannot be composed instead of fabricating week data', async () => {
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockRejectedValueOnce(new Error('weekday slots unavailable'));
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });
    requestMock.mockResolvedValueOnce({ items: [] });

    const appointments = await import('./appointments');

    await expect(
      appointments.listOperationalWeek({ professional_id: 'professional-1', date: '2026-04-08' }),
    ).rejects.toThrow('weekday slots unavailable');

		expect(requestMock).toHaveBeenCalledTimes(10);
	});

	it('fetches the new week agenda endpoint over authenticated transport', async () => {
		const weekAgenda: WeekAgenda = {
			professional_id: 'professional-1',
			week_start: '2026-04-06',
			templates: [{
				id: 'template-1',
				professional_id: 'professional-1',
				created_at: '2026-04-01T08:00:00Z',
				updated_at: '2026-04-01T08:00:00Z',
				versions: [{
					id: 'version-1',
					template_id: 'template-1',
					version_number: 1,
					effective_from: '2026-04-01T00:00:00Z',
					recurrence: {
						monday: { start_time: '09:00', end_time: '12:00', slot_duration_minutes: 30 },
					},
					created_at: '2026-04-01T08:00:00Z',
				}],
			}],
			blocks: [{
				id: 'block-1',
				professional_id: 'professional-1',
				scope: 'single',
				block_date: '2026-04-08T00:00:00Z',
				start_time: '10:00',
				end_time: '10:30',
				created_at: '2026-04-01T08:00:00Z',
				updated_at: '2026-04-01T08:00:00Z',
			}],
			consultations: [{
				id: 'consultation-1',
				slot_id: null,
				professional_id: 'professional-1',
				patient_id: 'patient-1',
				status: 'scheduled',
				source: 'doctor',
				scheduled_start: '2026-04-08T09:00:00Z',
				scheduled_end: '2026-04-08T09:20:00Z',
				created_at: '2026-04-01T08:00:00Z',
				updated_at: '2026-04-01T08:00:00Z',
			}],
			slots: [],
		};
		requestMock.mockResolvedValue(weekAgenda);

		const appointments = await import('./appointments');

		await expect(
			appointments.fetchWeekAgenda({ professional_id: 'professional-1', week_start: '2026-04-06' }),
		).resolves.toEqual(weekAgenda);

		expect(requestMock).toHaveBeenCalledWith('/appointments-api', '/agenda/week', {
			query: { professional_id: 'professional-1', week_start: '2026-04-06' },
			auth: true,
		});
	});

	it('surfaces week agenda transport failures without reshaping them', async () => {
		requestMock.mockRejectedValue(new Error('agenda unavailable'));

		const appointments = await import('./appointments');

		await expect(
			appointments.fetchWeekAgenda({ professional_id: 'professional-1', week_start: '2026-04-06' }),
		).rejects.toThrow('agenda unavailable');
	});
});
