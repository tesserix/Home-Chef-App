import { describe, expect, it } from 'vitest';

import { subscriptionRowSummary } from '../utils/subscription-summary-line';

describe('subscriptionRowSummary', () => {
  it('identifies itself as Tiffin while loading', () => {
    const summary = subscriptionRowSummary(true, false, null);

    expect(summary.startsWith('Tiffin')).toBe(true);
  });

  it('identifies itself as Tiffin on error, and keeps the tap-to-retry affordance', () => {
    const summary = subscriptionRowSummary(false, true, null);

    expect(summary.startsWith('Tiffin')).toBe(true);
    expect(summary).toMatch(/tap to view/i);
  });

  it('identifies itself as Tiffin when no meals are scheduled yet -- the default state every subscriber sees today (#907)', () => {
    const summary = subscriptionRowSummary(false, false, null);

    expect(summary).toBe('Tiffin · no meals scheduled yet');
  });

  it("identifies itself as Tiffin for today's meal", () => {
    const summary = subscriptionRowSummary(false, false, {
      day: { date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'scheduled' },
      isToday: true,
    });

    expect(summary).toBe('Tiffin · today: Lunch');
  });

  it('identifies itself as Tiffin for the next upcoming meal', () => {
    const summary = subscriptionRowSummary(false, false, {
      day: { date: '2026-08-05T06:30:00Z', slot: 'dinner', status: 'scheduled' },
      isToday: false,
    });

    expect(summary).toBe('Tiffin · next: Wed dinner');
    expect(summary).toMatch(/^Tiffin · next: \w{3} dinner$/);
  });

  it('every branch starts with the literal "Tiffin" -- the #907 regression this test guards against', () => {
    const branches = [
      subscriptionRowSummary(true, false, null),
      subscriptionRowSummary(false, true, null),
      subscriptionRowSummary(false, false, null),
      subscriptionRowSummary(false, false, {
        day: { date: '2026-08-02T06:30:00Z', slot: 'lunch', status: 'scheduled' },
        isToday: true,
      }),
      subscriptionRowSummary(false, false, {
        day: { date: '2026-08-05T06:30:00Z', slot: 'dinner', status: 'scheduled' },
        isToday: false,
      }),
    ];

    for (const summary of branches) {
      expect(summary.startsWith('Tiffin')).toBe(true);
    }
  });
});
