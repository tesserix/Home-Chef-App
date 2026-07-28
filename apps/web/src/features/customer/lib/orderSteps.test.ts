import { describe, expect, it } from 'vitest';
import {
  getChipLabel,
  getFeeRowLabel,
  getStatusLine,
  getStepIndex,
  getStepLabels,
  getStepStatuses,
  isPickupFulfillment,
} from './orderSteps';

// This model must match apps/mobile-customer/lib/orderSteps.ts exactly. A
// customer who ordered on the phone and opened the web app has to see the same
// journey — and the backend has no separate "collected" status, so the split
// lives entirely in these helpers.

describe('isPickupFulfillment', () => {
  it('treats an absent fulfillmentType as delivery', () => {
    // Orders placed before the column existed have it undefined. The server
    // defaults to 'delivery', so anything else here would relabel historical
    // delivery orders as pickup.
    expect(isPickupFulfillment(undefined)).toBe(false);
  });

  it('recognises pickup, and only pickup', () => {
    expect(isPickupFulfillment('pickup')).toBe(true);
    expect(isPickupFulfillment('delivery')).toBe(false);
    // chef_delivery is still someone bringing it to you.
    expect(isPickupFulfillment('chef_delivery')).toBe(false);
  });
});

describe('getStepLabels', () => {
  it('ends a pickup journey at Collected, not Delivered', () => {
    expect(getStepLabels('pickup')).toEqual([
      'Confirmed',
      'Preparing',
      'Ready for pickup',
      'Collected',
    ]);
  });

  it('keeps the carrier language for delivery and chef delivery', () => {
    const delivery = ['Confirmed', 'Preparing', 'On the way', 'Delivered'];
    expect(getStepLabels('delivery')).toEqual(delivery);
    expect(getStepLabels('chef_delivery')).toEqual(delivery);
    expect(getStepLabels(undefined)).toEqual(delivery);
  });
});

describe('getStepIndex', () => {
  it('advances pickup to step 2 at `ready` — that is the actionable moment', () => {
    // This is the pivot the whole module exists for: for pickup, `ready` means
    // "go and get it".
    expect(getStepIndex('ready', 'pickup')).toBe(2);
  });

  it('holds delivery at Preparing while `ready` waits for a carrier', () => {
    // The food being cooked is not the same as it being on its way — nobody has
    // collected it yet, so promising "On the way" would be a lie.
    expect(getStepIndex('ready', 'delivery')).toBe(1);
  });

  it('maps the shared early and terminal steps identically', () => {
    for (const mode of ['pickup', 'delivery'] as const) {
      expect(getStepIndex('accepted', mode)).toBe(0);
      expect(getStepIndex('preparing', mode)).toBe(1);
      expect(getStepIndex('delivered', mode)).toBe(3);
    }
  });

  it('returns -1 for statuses with no place on the bar', () => {
    expect(getStepIndex('pending', 'pickup')).toBe(-1);
    expect(getStepIndex('cancelled', 'delivery')).toBe(-1);
    expect(getStepIndex('refunded', 'delivery')).toBe(-1);
  });

  it('maps in-transit statuses to step 2 even on a pickup order', () => {
    // Shouldn't happen, but a defensive mapping beats an off-the-bar -1 that
    // would render a completed order as not started.
    expect(getStepIndex('picked_up', 'pickup')).toBe(2);
    expect(getStepIndex('delivering', 'delivery')).toBe(2);
  });
});

describe('getStepStatuses', () => {
  it('stays index-aligned with the labels', () => {
    for (const mode of ['pickup', 'delivery'] as const) {
      expect(getStepStatuses(mode)).toHaveLength(getStepLabels(mode).length);
    }
  });

  it('pivots on step 2: pickup waits at `ready`, delivery is `delivering`', () => {
    expect(getStepStatuses('pickup')[2]).toBe('ready');
    expect(getStepStatuses('delivery')[2]).toBe('delivering');
  });
});

describe('getStatusLine', () => {
  it('tells a pickup customer to go and collect', () => {
    expect(getStatusLine('ready', 'pickup')).toBe('Ready for pickup — collect from the chef');
  });

  it('never promises a driver on a delivery order', () => {
    // The customer doesn't choose the carrier — the chef does, at Mark Ready —
    // so the wording must stay neutral about who turns up.
    const line = getStatusLine('ready', 'delivery');
    expect(line).toBe('Almost ready — heading your way soon');
    expect(line).not.toMatch(/driver/i);
  });

  it('says Collected, not Delivered, once a pickup order is done', () => {
    expect(getStatusLine('delivered', 'pickup')).toBe('Collected');
    expect(getStatusLine('delivered', 'delivery')).toBe('Delivered');
  });
});

describe('getChipLabel', () => {
  it('distinguishes the two meanings of `ready`', () => {
    expect(getChipLabel('ready', 'pickup')).toBe('Ready for Pickup');
    expect(getChipLabel('ready', 'delivery')).toBe('Almost Ready');
  });

  it('relabels the terminal status for pickup', () => {
    expect(getChipLabel('delivered', 'pickup')).toBe('Collected');
    expect(getChipLabel('delivered', undefined)).toBe('Delivered');
  });

  it('shares the wording for statuses that mean the same thing either way', () => {
    for (const status of ['pending', 'accepted', 'preparing', 'cancelled', 'refunded'] as const) {
      expect(getChipLabel(status, 'pickup')).toBe(getChipLabel(status, 'delivery'));
    }
  });
});

describe('getFeeRowLabel', () => {
  it('never calls a pickup fee a delivery fee', () => {
    // Pickup is always free and nothing is delivered, so the row has to say so.
    expect(getFeeRowLabel('pickup')).toBe('Pickup');
    expect(getFeeRowLabel('delivery')).toBe('Delivery fee');
    expect(getFeeRowLabel('chef_delivery')).toBe('Delivery fee');
  });
});
