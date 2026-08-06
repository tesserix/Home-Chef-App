import { describe, expect, it } from '@jest/globals';

import { invalidationsFor, parseLiveFrame } from './live-updates';
import { streamBase } from '../hooks/useLiveUpdates';

describe('invalidationsFor', () => {
  it('refreshes the order lists for an order event', () => {
    expect(invalidationsFor({ order_id: 'o1' })).toEqual([
      ['notifications', 'unread'],
      ['notifications', 'list'],
      ['chef', 'orders'],
      ['chef', 'dashboard'],
      ['chef', 'upcoming'],
      ['chef', 'cancel-requests'],
    ]);
  });

  // Every frame we act on IS a notification that was just written. Without this the
  // orders list refreshed live while the bell badge kept its old count: a paid order
  // landed on the dashboard with the badge reading one short until something else
  // refetched it.
  it('refreshes the bell badge for every event', () => {
    for (const payload of [
      { order_id: 'o1' },
      { meal_plan_id: 'p1' },
      { meal_plan_day_id: 'd1' },
    ]) {
      expect(invalidationsFor(payload)).toContainEqual(['notifications', 'unread']);
      expect(invalidationsFor(payload)).toContainEqual(['notifications', 'list']);
    }
  });

  // A cancellation request arrives as an order event. The chef answers it in the
  // cancellations queue, so that queue must refresh on the same frame — without
  // this the order banner said "respond now" and the queue behind it was empty
  // for up to 30 s (#475).
  it('refreshes the cancellation queue for an order event', () => {
    expect(invalidationsFor({ order_id: 'o1' })).toContainEqual(['chef', 'cancel-requests']);
  });

  // The order-detail screen caches under ['chef','orders','detail-v2',id]. React
  // Query matches invalidations by PREFIX, so ['chef','orders'] has to be emitted
  // un-suffixed or an open order would never refetch and the stage button would
  // stay live under a pending cancellation.
  it('emits the order prefix that also matches the open detail screen', () => {
    expect(invalidationsFor({ order_id: 'o1' })).toContainEqual(['chef', 'orders']);
  });

  it('refreshes the plan lists for a meal-plan event', () => {
    const keys = invalidationsFor({ meal_plan_id: 'p1' });
    expect(keys).toContainEqual(['chef', 'meal-plans']);
    expect(keys).toContainEqual(['chef', 'prep']);
  });

  // A day-level event is a refund decision arriving in the chef's queue — the whole point
  // of the queue is that they act on it, so it must not wait for a poll.
  it('refreshes the refund queue for a day event', () => {
    const keys = invalidationsFor({ meal_plan_day_id: 'd1' });
    expect(keys).toContainEqual(['chef', 'refund-decisions']);
    expect(keys).toContainEqual(['chef', 'refunds']);
  });

  it('accepts day_id as well as meal_plan_day_id', () => {
    // Producers disagree on the field name; both mean the same thing to the chef.
    expect(invalidationsFor({ day_id: 'd1' })).toContainEqual(['chef', 'refund-decisions']);
  });

  // An event we have no view for must not refetch the world. The bell is the one
  // exception: invalidationsFor only ever sees a frame parseLiveFrame accepted, and
  // that is by definition a notification that was just written, so the badge is
  // stale either way. Two small queries, not the chef's whole dataset.
  it('refreshes only the bell for an event with nothing else we render', () => {
    expect(invalidationsFor({})).toEqual([
      ['notifications', 'unread'],
      ['notifications', 'list'],
    ]);
  });

  it('combines keys when one event touches a plan and its day', () => {
    const keys = invalidationsFor({ meal_plan_id: 'p1', meal_plan_day_id: 'd1' });
    expect(keys).toContainEqual(['chef', 'meal-plans']);
    expect(keys).toContainEqual(['chef', 'refund-decisions']);
  });
});

describe('parseLiveFrame', () => {
  it('reads the payload out of a notification frame', () => {
    const raw = JSON.stringify({ type: 'new_notification', data: JSON.stringify({ order_id: 'o1' }) });
    expect(parseLiveFrame(raw)).toEqual({ order_id: 'o1' });
  });

  // The bell's badge frame is not an update; acting on it would refetch on every unread tick.
  it('ignores the unread-count frame', () => {
    expect(parseLiveFrame(JSON.stringify({ type: 'unread_count', unreadCount: 3 }))).toBeNull();
  });

  it('survives malformed frames rather than throwing into the socket handler', () => {
    expect(parseLiveFrame('not json')).toBeNull();
    expect(parseLiveFrame(JSON.stringify({ type: 'new_notification' }))).toBeNull();
    expect(parseLiveFrame(JSON.stringify({ type: 'new_notification', data: '{oops' }))).toBeNull();
  });
});

// The apps disagree about where /v1 lives: the vendor base ends in /api/v1, the customer's
// in /api. Appending it blindly produced /api/v1/v1/... — a 404 whose only symptom was
// stale data, because the stream fails silently.
describe('streamBase', () => {
  it('does not double a /v1 the base already carries', () => {
    expect(streamBase('https://vendors.fe3dr.com/api/v1').http).toBe(
      'https://vendors.fe3dr.com/api/v1',
    );
  });

  it('adds /v1 when the base stops at /api', () => {
    expect(streamBase('https://fe3dr.com/api').http).toBe('https://fe3dr.com/api/v1');
  });

  it('tolerates a trailing slash', () => {
    expect(streamBase('https://fe3dr.com/api/v1/').http).toBe('https://fe3dr.com/api/v1');
  });

  it('leaves a plaintext base alone (local dev)', () => {
    expect(streamBase('http://localhost:8090/api/v1').http).toBe('http://localhost:8090/api/v1');
  });
});
