import { describe, it, expect } from '@jest/globals';

import {
  mealPlanStatusMeta,
  mealPlanDayStatusMeta,
  pickLiveMealPlan,
  isLiveMealPlanStatus,
  isDeclinedDayStatus,
  summarizeLivePlan,
  toLocalDateKey,
} from './meal-plan';
import type { MealPlan, MealPlanDay } from '../hooks/useMealPlans';

describe('mealPlanDayStatusMeta', () => {
  // Every day status the plan lifecycle can produce must resolve to a labeled
  // pill — the sheet + detail list show a pill for all of them, not just the
  // live prepared/delivered ones.
  const statuses = [
    'requested',
    'accepted',
    'confirmed',
    'prepared',
    'delivered',
    'declined',
    'skipped',
    'cancelled',
    'refunded',
  ];

  it('returns a non-empty label for every day status', () => {
    for (const s of statuses) {
      expect(mealPlanDayStatusMeta(s).label.trim().length).toBeGreaterThan(0);
    }
  });

  it('flags only "prepared" as cooking (drives the animated indicator)', () => {
    expect(mealPlanDayStatusMeta('prepared').cooking).toBe(true);
    expect(mealPlanDayStatusMeta('delivered').cooking).toBe(false);
    expect(mealPlanDayStatusMeta('confirmed').cooking).toBe(false);
  });

  it('labels scheduled/skipped/refunded readably', () => {
    expect(mealPlanDayStatusMeta('confirmed').label).toBe('Scheduled');
    expect(mealPlanDayStatusMeta('skipped').label).toBe('Skipped');
    expect(mealPlanDayStatusMeta('refunded').label).toBe('Refunded');
  });
});

describe('mealPlanStatusMeta', () => {
  it('marks awaiting_customer / chef_modified as needing action', () => {
    expect(mealPlanStatusMeta('awaiting_customer').needsAction).toBe(true);
    expect(mealPlanStatusMeta('chef_modified').needsAction).toBe(true);
    expect(mealPlanStatusMeta('confirmed').needsAction).toBe(false);
  });
});

describe('isLiveMealPlanStatus', () => {
  it('is true for in-flight/running statuses, false for terminal', () => {
    ['pending_chef', 'awaiting_customer', 'chef_modified', 'chef_accepted_full', 'confirmed', 'active'].forEach(
      (s) => expect(isLiveMealPlanStatus(s)).toBe(true),
    );
    ['completed', 'cancelled', 'expired', 'unknown'].forEach((s) =>
      expect(isLiveMealPlanStatus(s)).toBe(false),
    );
  });
});

describe('isDeclinedDayStatus', () => {
  it('is true for won\'t-be-served statuses, false otherwise', () => {
    ['declined', 'skipped', 'cancelled', 'refunded'].forEach((s) =>
      expect(isDeclinedDayStatus(s)).toBe(true),
    );
    ['confirmed', 'prepared', 'delivered', 'requested'].forEach((s) =>
      expect(isDeclinedDayStatus(s)).toBe(false),
    );
  });
});

describe('summarizeLivePlan', () => {
  const day = (over: Partial<MealPlanDay>): MealPlanDay =>
    ({ id: 'd', date: '2026-07-05', slot: 'lunch', variant: 'veg', status: 'confirmed', price: 100, ...over }) as MealPlanDay;

  it('counts days still to be served (excludes delivered + declined-ish)', () => {
    const days = [
      day({ date: '2026-07-01', status: 'delivered' }),
      day({ date: '2026-07-02', status: 'skipped' }),
      day({ date: '2026-07-03', status: 'confirmed' }),
      day({ date: '2026-07-04', status: 'prepared' }),
    ];
    expect(summarizeLivePlan(days, '2026-07-04').daysLeft).toBe(2);
  });

  it('reports today\'s status when a day falls on today, else null', () => {
    const days = [day({ date: '2026-07-05', status: 'prepared' })];
    expect(summarizeLivePlan(days, '2026-07-05').todayStatus).toBe('prepared');
    expect(summarizeLivePlan(days, '2026-07-06').todayStatus).toBeNull();
  });

  it('matches the API RFC3339 timestamp shape, not just YYYY-MM-DD', () => {
    // MealPlanDay.date arrives as a full instant; the today-match must normalize
    // it to a local calendar day. Compute the expected key the same way so the
    // assertion is timezone-independent.
    const iso = '2026-07-05T00:00:00Z';
    const days = [day({ date: iso, status: 'prepared' })];
    const key = toLocalDateKey(iso);
    expect(summarizeLivePlan(days, key).todayStatus).toBe('prepared');
  });

  it('handles an empty plan', () => {
    expect(summarizeLivePlan([], '2026-07-05')).toEqual({ daysLeft: 0, todayStatus: null });
  });
});

describe('pickLiveMealPlan', () => {
  const mk = (over: Partial<MealPlan>): MealPlan =>
    ({
      id: 'x',
      chefId: 'c1',
      status: 'confirmed',
      startDate: '2026-07-01',
      endDate: '2026-07-07',
      days: [],
      ...over,
    }) as MealPlan;

  it('returns undefined when there are no plans', () => {
    expect(pickLiveMealPlan([])).toBeUndefined();
    expect(pickLiveMealPlan(undefined)).toBeUndefined();
  });

  it('ignores terminal plans', () => {
    expect(pickLiveMealPlan([mk({ status: 'completed' }), mk({ status: 'cancelled' })])).toBeUndefined();
  });

  it('returns the most recent live plan by startDate', () => {
    const older = mk({ id: 'old', startDate: '2026-06-01' });
    const newer = mk({ id: 'new', startDate: '2026-07-10' });
    expect(pickLiveMealPlan([older, newer])?.id).toBe('new');
  });

  it('scopes to a chef when chefId is given', () => {
    const mine = mk({ id: 'mine', chefId: 'c1', startDate: '2026-06-01' });
    const other = mk({ id: 'other', chefId: 'c2', startDate: '2026-07-10' });
    expect(pickLiveMealPlan([mine, other], 'c1')?.id).toBe('mine');
    expect(pickLiveMealPlan([other], 'c1')).toBeUndefined();
  });
});
