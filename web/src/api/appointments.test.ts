import { beforeEach, describe, expect, it, vi } from 'vitest';

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
});
