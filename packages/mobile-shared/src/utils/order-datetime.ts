// When an order was placed, rendered the same way on every surface.
//
// Each screen had grown its own answer: the customer card showed a date with no
// time, the vendor history row showed only "2h ago", and the vendor order detail
// showed nothing at all. A chef reconciling a day's cooking, or a customer
// checking which of two orders was which, could not tell from the screen.
//
// Relative time ("2h ago") is still right for a live queue — it answers "is this
// urgent?". It is wrong as the only answer to "when was this placed?", which is
// what a history row and a detail screen are for.

/**
 * Absolute placement time, e.g. `4 Aug 2026, 10:18 am`.
 *
 * en-IN so the day leads the month — a dd/mm audience reading mm/dd silently
 * mis-reads every date in the first twelve days of a month.
 */
export function formatOrderDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

/** Same, without the year — for rows already grouped under a date header. */
export function formatOrderTimeShort(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    hour: 'numeric',
    minute: '2-digit',
  });
}
