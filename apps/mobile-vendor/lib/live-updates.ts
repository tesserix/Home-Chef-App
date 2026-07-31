// live-updates.ts — which caches a live event invalidates.
//
// Kept pure and separate from the transport so the routing is unit-testable and so WS and
// SSE (or any future transport) cannot drift apart: they both hand a payload to this.
//
// The stream is user-scoped, so anything arriving here is already this chef's business —
// the only question is which lists it changes.

/** The shape the server puts in a notification's `data` blob. Everything is optional: the
 *  payloads come from a dozen producers and a missing field must never throw. */
export interface LiveEventPayload {
  order_id?: string;
  meal_plan_id?: string;
  meal_plan_day_id?: string;
  day_id?: string;
}

// The chef's own views. Grouped by what a change actually invalidates rather than by which
// event produced it, because several events land on the same lists.
const ORDER_KEYS = [
  ['chef', 'orders'],
  ['chef', 'dashboard'],
  ['chef', 'upcoming'],
];
const MEAL_PLAN_KEYS = [
  ['chef', 'meal-plans'],
  ['chef', 'prep'],
  ['chef', 'dashboard'],
];
// A day-level event is a refund decision or a skip landing in the chef's queue.
const REFUND_KEYS = [
  ['chef', 'refund-decisions'],
  ['chef', 'refunds'],
];

/**
 * The query keys a live event should invalidate. Returns an empty array for anything
 * unrecognised — an event we have no view for is not an error, and refetching the whole
 * app because a loyalty point moved would be worse than ignoring it.
 */
export function invalidationsFor(payload: LiveEventPayload): string[][] {
  const keys: string[][] = [];
  if (payload.order_id) {
    keys.push(...ORDER_KEYS);
  }
  if (payload.meal_plan_id) {
    keys.push(...MEAL_PLAN_KEYS);
  }
  // A day carries its own refund lifecycle, and the chef has a queue for exactly that.
  if (payload.meal_plan_day_id || payload.day_id) {
    keys.push(...REFUND_KEYS);
  }
  return keys;
}

/** Parses a raw stream frame into a payload, or null if it is not one we act on. */
export function parseLiveFrame(raw: string): LiveEventPayload | null {
  try {
    const msg = JSON.parse(raw) as { type?: string; data?: string };
    // The bell's own unread-count frames carry no data blob — not an update, just a badge.
    if (msg.type !== 'new_notification' || !msg.data) return null;
    return JSON.parse(msg.data) as LiveEventPayload;
  } catch {
    return null;
  }
}
