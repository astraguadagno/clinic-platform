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
});
