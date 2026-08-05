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
  fssai_request_id?: string;
}

// The chef's own views. Grouped by what a change actually invalidates rather than by which
// event produced it, because several events land on the same lists.
// ['chef','orders'] is a PREFIX: React Query matches it against
// ['chef','orders','detail-v2',<id>] too, so an open order-detail screen
// refetches on the same event as the list. That is what makes a cancellation
// request grey out the stage button while the chef is looking at the order,
// rather than on their next pull-to-refresh (#475).
const ORDER_KEYS = [
  ['chef', 'orders'],
  ['chef', 'dashboard'],
  ['chef', 'upcoming'],
  // The cancellation queue is where the chef answers that request. Leaving it
  // off meant the banner said "respond now" and the queue behind it was still
  // up to 30 s stale.
  ['chef', 'cancel-requests'],
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
// Every frame we act on IS a notification that was just written, so the bell's
// badge and list are stale the moment it arrives. Without these the orders list
// refreshed live while the badge sat on its old count until something else
// refetched it — a paid order landed with the bell still reading one short.
const NOTIFICATION_KEYS = [
  ['notifications', 'unread'],
  ['notifications', 'list'],
];

/**
 * The query keys a live event should invalidate. Returns an empty array for anything
 * unrecognised — an event we have no view for is not an error, and refetching the whole
 * app because a loyalty point moved would be worse than ignoring it.
 */
// The filing request and the dashboard card that mirrors it. A status an admin
// sets is meant to reach the chef "straight away" — the admin screen says so —
// and without this it waited for a refetch.
const FSSAI_KEYS = [['chef', 'fssai', 'request']];

export function invalidationsFor(payload: LiveEventPayload): string[][] {
  const keys: string[][] = [...NOTIFICATION_KEYS];
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
  if (payload.fssai_request_id) {
    keys.push(...FSSAI_KEYS);
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
