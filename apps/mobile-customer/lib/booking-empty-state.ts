/**
 * Which empty state (if any) the tiffin booking screen should show.
 *
 * A chef can publish a fixed weekly menu, per-date menus (#405 — the only ones
 * that carry combos/thalis), or both. The screen used to gate purely on the
 * weekly menu, so a chef cooking day-by-day was reported as having published
 * nothing even though their per-date menu had already been fetched and turned
 * into bookable dates.
 *
 * Bookable dates are therefore the authority: if there is something to book,
 * show it, whichever menu produced it.
 */
export type BookingEmptyState = 'no-menu' | 'no-dates' | null;

export function bookingEmptyState(
  weeklyPublished: boolean,
  bookableDateCount: number,
): BookingEmptyState {
  if (bookableDateCount > 0) return null;
  return weeklyPublished ? 'no-dates' : 'no-menu';
}
