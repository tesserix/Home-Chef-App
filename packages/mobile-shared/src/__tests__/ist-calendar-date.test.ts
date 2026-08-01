import { describe, expect, it } from 'vitest';

import { isSkippableMealDay, istCalendarDate } from '../utils/ist-calendar-date';

describe('istCalendarDate / isSkippableMealDay', () => {
  it('device already in IST, mid-afternoon: tomorrow (IST) is skippable, today (IST) is not', () => {
    // 2026-08-02 10:00 UTC === 2026-08-02 15:30 IST (mid-afternoon)
    const now = new Date('2026-08-02T10:00:00.000Z');
    expect(isSkippableMealDay('2026-08-03', now)).toBe(true);
    expect(isSkippableMealDay('2026-08-02', now)).toBe(false);
  });

  it('a device physically in UTC-8 depends on the instant, never on device-local calendar day', () => {
    // Local wall-clock at UTC-8 reads "Aug 2, 23:00" -> absolute instant Aug 3 07:00 UTC,
    // which is objectively Aug 3 12:30 IST.
    const now = new Date(Date.UTC(2026, 7, 3, 7, 0, 0));
    expect(istCalendarDate(now)).toBe('2026-08-03');
  });

  it('one millisecond before IST midnight: the next IST day is still skippable', () => {
    const now = new Date('2026-08-02T18:29:59.999Z');
    // 2026-08-02T18:30:00.000Z is IST midnight of 2026-08-03 -- the "next" IST day.
    expect(isSkippableMealDay('2026-08-02T18:30:00.000Z', now)).toBe(true);
  });

  it('exactly at IST midnight: that instant is now "today" and is not skippable', () => {
    const now = new Date('2026-08-02T18:30:00.000Z');
    expect(isSkippableMealDay('2026-08-02T18:30:00.000Z', now)).toBe(false);
  });

  it('a past date relative to now is not skippable', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    expect(isSkippableMealDay('2026-08-01', now)).toBe(false);
  });

  it('a bare YYYY-MM-DD dateISO (no time/offset) still resolves correctly', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    expect(isSkippableMealDay('2026-08-03', now)).toBe(true);
    expect(isSkippableMealDay('2026-08-02', now)).toBe(false);
    expect(isSkippableMealDay('2026-08-01', now)).toBe(false);
  });
});
