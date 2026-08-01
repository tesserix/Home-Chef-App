import { describe, expect, it } from 'vitest';

import { pickSubscriptionMealDay } from '../utils/subscription-meal-day';

describe('pickSubscriptionMealDay', () => {
  it('picks today when a scheduled day matches the IST calendar day of now', () => {
    // 2026-08-02 10:00 UTC === 2026-08-02 15:30 IST (mid-afternoon)
    const now = new Date('2026-08-02T10:00:00.000Z');
    const days = [{ date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'scheduled' }];

    const picked = pickSubscriptionMealDay(days, now);

    expect(picked).not.toBeNull();
    expect(picked?.day).toBe(days[0]);
    expect(picked?.isToday).toBe(true);
  });

  it('falls through to a later-this-week scheduled day when nothing is scheduled today', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    const laterDay = { date: '2026-08-05T06:30:00Z', slot: 'dinner', status: 'scheduled' };
    const days = [
      { date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'missed' },
      laterDay,
    ];

    const picked = pickSubscriptionMealDay(days, now);

    expect(picked?.day).toBe(laterDay);
    expect(picked?.isToday).toBe(false);
  });

  it('returns null when there are no days at all', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');

    expect(pickSubscriptionMealDay([], now)).toBeNull();
  });

  it('returns null when every day is strictly before today (IST), even if scheduled', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    const days = [
      { date: '2026-08-01T06:30:00Z', slot: 'lunch', status: 'scheduled' },
      { date: '2026-07-31T06:30:00Z', slot: 'dinner', status: 'scheduled' },
    ];

    expect(pickSubscriptionMealDay(days, now)).toBeNull();
  });

  it('tie-breaks same-day entries so lunch sorts before dinner', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    const lunch = { date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'scheduled' };
    const dinner = { date: '2026-08-02T12:30:00Z', slot: 'dinner', status: 'scheduled' };
    const days = [dinner, lunch];

    const picked = pickSubscriptionMealDay(days, now);

    expect(picked?.day).toBe(lunch);
    expect(picked?.isToday).toBe(true);
  });

  it('a skipped today does not mask a genuinely upcoming scheduled day', () => {
    const now = new Date('2026-08-02T10:00:00.000Z');
    const laterDay = { date: '2026-08-04T06:30:00Z', slot: 'lunch', status: 'scheduled' };
    const days = [
      { date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'skipped' },
      laterDay,
    ];

    const picked = pickSubscriptionMealDay(days, now);

    expect(picked?.day).toBe(laterDay);
    expect(picked?.isToday).toBe(false);
  });
});
