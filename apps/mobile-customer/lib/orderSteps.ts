// Fulfillment-aware order lifecycle steps for the customer app.
//
// Delivery and pickup are different journeys, so the progress stepper, the
// timeline, and the friendly status sentences must differ:
//
//   delivery / chef_delivery : Confirmed → Preparing → On the way → Delivered
//   pickup                   : Confirmed → Preparing → Ready for pickup → Collected
//
// The key difference is the `ready` status: for delivery it still reads
// "Preparing" (the food just waits for the driver), but for pickup it is the
// actionable moment ("Ready for pickup — go collect it"). The terminal
// `delivered` status is relabeled "Collected" for pickup; there is no separate
// backend status (see apps/api/handlers/chefs.go UpdateOrderStatus).
//
// Centralizing this here keeps OrderProgressBar, OrderTimeline, ActiveOrderCard,
// track.tsx, and the order-detail screen in lock-step instead of each hardcoding
// its own delivery-only mapping.

import type { Order } from '../types/customer';

export type OrderStatus = Order['status'];
export type FulfillmentType = NonNullable<Order['fulfillmentType']>;

/** True for pickup orders — the customer collects from the chef. */
export function isPickupFulfillment(
  fulfillment: Order['fulfillmentType'],
): boolean {
  return fulfillment === 'pickup';
}

const DELIVERY_STEPS = [
  'Confirmed',
  'Preparing',
  'On the way',
  'Delivered',
] as const;

const PICKUP_STEPS = [
  'Confirmed',
  'Preparing',
  'Ready for pickup',
  'Collected',
] as const;

/** The four step labels for the given fulfillment type. */
export function getStepLabels(
  fulfillment: Order['fulfillmentType'],
): readonly string[] {
  return isPickupFulfillment(fulfillment) ? PICKUP_STEPS : DELIVERY_STEPS;
}

/**
 * The active step index (0-based) for a status. Returns -1 for statuses with
 * no place on the bar (pending before confirm, cancelled, refunded).
 */
export function getStepIndex(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType'],
): number {
  const pickup = isPickupFulfillment(fulfillment);
  switch (status) {
    case 'pending':
      return -1;
    case 'accepted':
      return 0;
    case 'preparing':
      return 1;
    case 'ready':
      // The pivot: pickup advances to "Ready for pickup" (step 2); delivery
      // stays at "Preparing" (step 1) until a driver picks it up.
      return pickup ? 2 : 1;
    case 'picked_up':
    case 'delivering':
      // Shouldn't occur for pickup, but map defensively to the 3rd step.
      return 2;
    case 'delivered':
      return 3;
    default:
      return -1;
  }
}

/**
 * A friendly status sentence for the active-order card and inline status rows.
 * Pickup wording never mentions a driver or delivery.
 */
export function getStatusLine(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType'],
): string {
  const pickup = isPickupFulfillment(fulfillment);
  switch (status) {
    case 'pending':
      return 'Order received';
    case 'accepted':
      return 'Order confirmed';
    case 'preparing':
      return 'Chef is preparing your order';
    case 'ready':
      // Delivery wording stays neutral about WHO carries it (chef vs 3PL) — the
      // customer doesn't choose that, so never promise a "driver".
      return pickup
        ? 'Ready for pickup — collect from the chef'
        : 'Almost ready — heading your way soon';
    case 'picked_up':
      return 'On the way to you';
    case 'delivering':
      return 'Out for delivery';
    case 'delivered':
      return pickup ? 'Collected' : 'Delivered';
    case 'cancelled':
      return 'Cancelled';
    case 'refunded':
      return 'Refunded';
    default:
      return status;
  }
}

/** Short chip label (Title Case) for the status chip on the detail screen. */
export function getChipLabel(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType'],
): string {
  const pickup = isPickupFulfillment(fulfillment);
  switch (status) {
    case 'pending':
      return 'Pending';
    case 'accepted':
      return 'Confirmed';
    case 'preparing':
      return 'Preparing';
    case 'ready':
      return pickup ? 'Ready for Pickup' : 'Almost Ready';
    case 'picked_up':
      return 'On the Way';
    case 'delivering':
      return 'Out for Delivery';
    case 'delivered':
      return pickup ? 'Collected' : 'Delivered';
    case 'cancelled':
      return 'Cancelled';
    case 'refunded':
      return 'Refunded';
    default:
      return status;
  }
}
