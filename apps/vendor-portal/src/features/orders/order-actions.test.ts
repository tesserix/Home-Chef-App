import { describe, it, expect } from 'vitest';
import { getChefOrderAction, isLiveChefOrder } from './order-actions';

describe('getChefOrderAction', () => {
  it('gates Mark Ready behind the food-ready photo', () => {
    const action = getChefOrderAction('preparing', 'delivery');
    expect(action).toMatchObject({
      kind: 'advance',
      nextStatus: 'ready',
      photoKind: 'ready',
    });
  });

  it('gates Mark Ready behind a photo on pickup orders too, with pickup wording', () => {
    const action = getChefOrderAction('preparing', 'pickup');
    expect(action).toMatchObject({
      kind: 'advance',
      label: 'Mark ready for pickup',
      nextStatus: 'ready',
      photoKind: 'ready',
    });
  });

  it('gates the pickup handover behind the proof-of-handover photo', () => {
    const action = getChefOrderAction('ready', 'pickup');
    expect(action).toMatchObject({
      kind: 'advance',
      label: 'Mark handed over',
      nextStatus: 'delivered',
      photoKind: 'handover',
    });
  });

  it('commits self-delivery at Mark Ready when no rider can be dispatched', () => {
    // Otherwise the order lands at `ready` on a 3PL route with no 3PL — the
    // dead-end this whole flow exists to avoid.
    expect(
      getChefOrderAction('preparing', 'delivery', {
        offersSelfDelivery: true,
        riderDispatchAvailable: false,
      })
    ).toMatchObject({
      label: "Mark Ready · I'll deliver",
      photoKind: 'ready',
      carrier: 'chef_delivery',
    });
  });

  it('leaves Mark Ready on the rider route while 3PL is live', () => {
    expect(
      getChefOrderAction('preparing', 'delivery', {
        offersSelfDelivery: true,
        riderDispatchAvailable: true,
      })
    ).not.toHaveProperty('carrier');
  });

  it('never commits a carrier on a pickup order', () => {
    expect(
      getChefOrderAction('preparing', 'pickup', {
        offersSelfDelivery: true,
        riderDispatchAvailable: false,
      })
    ).not.toHaveProperty('carrier');
  });

  it('needs no photo to start preparing', () => {
    expect(getChefOrderAction('accepted', 'delivery')).toMatchObject({
      kind: 'advance',
      nextStatus: 'preparing',
      photoKind: null,
    });
  });

  it('leaves a ready 3PL delivery order waiting on a rider, with no chef action', () => {
    expect(
      getChefOrderAction('ready', 'delivery', { riderDispatchAvailable: true })
    ).toEqual({ kind: 'waiting', caption: 'Waiting for a rider to pick up' });
  });

  it('treats a missing fulfillmentType as delivery (legacy rows)', () => {
    expect(getChefOrderAction('ready')).toEqual(getChefOrderAction('ready', 'delivery'));
  });

  // The dead-end that stranded order HC26072808359105: fulfillment_type
  // 'delivery', status 'ready', and every 3PL provider disabled. Nothing was
  // ever going to collect it, and neither chef surface offered a way out.
  it('offers self-delivery as the only route when no rider can be dispatched', () => {
    const action = getChefOrderAction('ready', 'delivery', {
      offersSelfDelivery: true,
      riderDispatchAvailable: false,
    });
    expect(action).toEqual({
      kind: 'waiting',
      caption: 'No delivery partner is available',
      switchTo: {
        label: "I'll deliver this instead",
        carrier: 'chef_delivery',
        hint: 'Deliver it yourself to complete this order.',
      },
    });
  });

  it('offers the self-delivery switch without the urgency hint while riders are live', () => {
    const action = getChefOrderAction('ready', 'delivery', {
      offersSelfDelivery: true,
      riderDispatchAvailable: true,
    });
    expect(action).toMatchObject({
      caption: 'Waiting for a rider to pick up',
      switchTo: { carrier: 'chef_delivery' },
    });
    expect(action?.kind === 'waiting' && action.switchTo?.hint).toBeUndefined();
  });

  it('offers no switch to a chef who does not self-deliver', () => {
    const action = getChefOrderAction('ready', 'delivery', {
      offersSelfDelivery: false,
      riderDispatchAvailable: false,
    });
    expect(action).toEqual({ kind: 'waiting', caption: 'No delivery partner is available' });
  });

  it('lets a self-delivering chef hand back to a rider only while 3PL is live', () => {
    expect(
      getChefOrderAction('ready', 'chef_delivery', { riderDispatchAvailable: true })
    ).toMatchObject({ switchTo: { carrier: 'delivery' } });
    expect(
      getChefOrderAction('ready', 'chef_delivery', { riderDispatchAvailable: false })
    ).not.toHaveProperty('switchTo');
  });

  it('never offers a carrier switch on a pickup order — the API rejects it', () => {
    const action = getChefOrderAction('ready', 'pickup', {
      offersSelfDelivery: true,
      riderDispatchAvailable: true,
    });
    expect(action).not.toHaveProperty('switchTo');
  });

  it('walks a self-delivering chef out and back: ready → picked_up → delivered', () => {
    expect(getChefOrderAction('ready', 'chef_delivery')).toMatchObject({
      kind: 'advance',
      nextStatus: 'picked_up',
      photoKind: null,
    });
    expect(getChefOrderAction('picked_up', 'chef_delivery')).toMatchObject({
      kind: 'advance',
      nextStatus: 'delivered',
      photoKind: null,
    });
  });

  it('offers nothing on terminal statuses', () => {
    expect(getChefOrderAction('delivered', 'pickup')).toBeNull();
    expect(getChefOrderAction('cancelled', 'delivery')).toBeNull();
    expect(getChefOrderAction('rejected', 'delivery')).toBeNull();
  });

  it('never advances a pending order — accept/reject is a separate decision', () => {
    expect(getChefOrderAction('pending', 'delivery')).toBeNull();
  });
});

describe('isLiveChefOrder', () => {
  it('keeps a self-delivering chef en route in the live queue', () => {
    expect(isLiveChefOrder({ status: 'picked_up', fulfillmentType: 'chef_delivery' })).toBe(true);
  });

  it('drops a picked-up 3PL order — the rider owns it now', () => {
    expect(isLiveChefOrder({ status: 'picked_up', fulfillmentType: 'delivery' })).toBe(false);
    expect(isLiveChefOrder({ status: 'picked_up' })).toBe(false);
  });

  it('drops any order under an open delivery-failure review', () => {
    expect(
      isLiveChefOrder({
        status: 'picked_up',
        fulfillmentType: 'chef_delivery',
        deliveryFailureReported: true,
      })
    ).toBe(false);
  });

  it('keeps everything still in the kitchen', () => {
    for (const status of ['pending', 'accepted', 'preparing', 'ready'] as const) {
      expect(isLiveChefOrder({ status })).toBe(true);
    }
  });
});
