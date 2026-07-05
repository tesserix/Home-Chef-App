import { describe, it, expect } from '@jest/globals';

import { orderSourceLabel, orderSourceBadge } from './orderSource';

describe('orderSourceLabel', () => {
  it('labels the non-à-la-carte sources', () => {
    expect(orderSourceLabel('meal_plan')).toBe('Meal plan');
    expect(orderSourceLabel('subscription')).toBe('Subscription');
    expect(orderSourceLabel('group')).toBe('Group');
  });

  it('returns null for à-la-carte, undefined, and unknown (no badge)', () => {
    expect(orderSourceLabel('alacarte')).toBeNull();
    expect(orderSourceLabel(undefined)).toBeNull();
    expect(orderSourceLabel('something-new')).toBeNull();
  });
});

describe('orderSourceBadge', () => {
  it('returns a label + colors for a tagged source', () => {
    const b = orderSourceBadge('meal_plan');
    expect(b?.label).toBe('Meal plan');
    expect(typeof b?.color).toBe('string');
    expect(typeof b?.bg).toBe('string');
  });

  it('returns null for à-la-carte (no badge)', () => {
    expect(orderSourceBadge('alacarte')).toBeNull();
    expect(orderSourceBadge(undefined)).toBeNull();
  });
});
