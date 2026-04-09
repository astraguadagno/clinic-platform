import { describe, expect, it } from 'vitest';
import {
  formatWeekdayLabel,
  getOperationalWeekDays,
  getUtcWeekStart,
  mapDateToOperationalWeek,
  normalizeOperationalTimeBands,
} from './helpers';

describe('schedule week helpers', () => {
  it('derives the monday week start in UTC from any selected day', () => {
    expect(getUtcWeekStart('2026-04-08')).toBe('2026-04-06');
    expect(getUtcWeekStart('2026-04-12')).toBe('2026-04-06');
  });

  it('maps any selected day to a visible monday-friday operational week', () => {
    expect(getOperationalWeekDays('2026-04-06').map((day) => day.date)).toEqual([
      '2026-04-06',
      '2026-04-07',
      '2026-04-08',
      '2026-04-09',
      '2026-04-10',
    ]);
    expect(mapDateToOperationalWeek('2026-04-12')).toBe('2026-04-10');
  });

  it('formats short weekday labels and normalizes sparse time bands', () => {
    expect(formatWeekdayLabel('2026-04-06')).toBe('Lun 06/04');
    expect(
      normalizeOperationalTimeBands([
        { start_time: '2026-04-06T10:00:00Z' },
        { start_time: '2026-04-07T09:00:00Z' },
        { start_time: '2026-04-06T10:00:00Z' },
      ]),
    ).toEqual(['09:00', '10:00']);
  });
});
