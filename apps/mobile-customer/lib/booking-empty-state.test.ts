import { describe, it, expect } from '@jest/globals';

import { bookingEmptyState } from './booking-empty-state';

// book-meal-plan already fetches the chef's per-date menus and the `dates` memo
// already prefers them over the weekly template ("dynamic dishes + combos,
// #405/#406"). But the render short-circuited on `!menu?.isPublished` — the
// WEEKLY menu — so a chef who publishes only per-date menus hit "No weekly menu
// yet" and the daily data that had already been fetched was never rendered.
//
// That is the only surface carrying combos, so a published thali was invisible
// to customers while the vendor app said "Published — customers can book this
// day".

describe('bookingEmptyState', () => {
  it('shows nothing to book when the chef has neither menu', () => {
    expect(bookingEmptyState(false, 0)).toBe('no-menu');
  });

  it('renders the grid for a daily-only chef', () => {
    // The regression: weekly menu absent, but per-date menus produced dates.
    expect(bookingEmptyState(false, 5)).toBeNull();
  });

  it('renders the grid for a weekly-only chef', () => {
    expect(bookingEmptyState(true, 5)).toBeNull();
  });

  it('reports no upcoming days when a weekly menu exists but every date is past cutoff', () => {
    expect(bookingEmptyState(true, 0)).toBe('no-dates');
  });
});
