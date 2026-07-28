// Fulfillment-aware order lifecycle steps for the customer web app.
//
// The web mirror of `apps/mobile-customer/lib/orderSteps.ts`. Delivery and
// pickup are different journeys, so the progress stepper and the status wording
// must differ:
//
//   delivery / chef_delivery : Confirmed → Preparing → On the way → Delivered
//   pickup                   : Confirmed → Preparing → Ready for pickup → Collected
//
// The key difference is the `ready` status: for delivery it still reads
// "Preparing" (the food just waits for a carrier), but for pickup it is the
// actionable moment ("Ready for pickup — go collect it"). The terminal
// `delivered` status is relabeled "Collected" for pickup; there is no separate
// backend status (see apps/api/handlers/chefs.go UpdateOrderStatus).
//
// Web previously hardcoded the delivery-only ladder in OrderDetailPage, so a
// pickup order told the customer their food was "On the way" when in fact it was
// sitting on the chef's counter waiting for them. Centralizing here keeps every
// web surface in lock-step with mobile instead of each re-deriving the split.

import type { FulfillmentType, Order, OrderStatus } from '@/shared/types';

/** True for pickup orders — the customer collects from the chef. */
export function isPickupFulfillment(fulfillment: Order['fulfillmentType']): boolean {
  return fulfillment === 'pickup';
}

const DELIVERY_STEPS = ['Confirmed', 'Preparing', 'On the way', 'Delivered'] as const;

const PICKUP_STEPS = ['Confirmed', 'Preparing', 'Ready for pickup', 'Collected'] as const;

/** The four step labels for the given fulfillment type. */
export function getStepLabels(fulfillment: Order['fulfillmentType']): readonly string[] {
  return isPickupFulfillment(fulfillment) ? PICKUP_STEPS : DELIVERY_STEPS;
}

// The status each step represents, so anything rendering a per-step icon derives
// it from here instead of re-deriving the delivery/pickup split. Index-aligned
// with getStepLabels.
//
// Step 2 is the pivot, mirroring getStepIndex: delivery is in motion
// ("On the way" → delivering), pickup is waiting to be collected
// ("Ready for pickup" → ready).
const DELIVERY_STEP_STATUSES = [
  'accepted',
  'preparing',
  'delivering',
  'delivered',
] as const satisfies readonly OrderStatus[];

const PICKUP_STEP_STATUSES = [
  'accepted',
  'preparing',
  'ready',
  'delivered',
] as const satisfies readonly OrderStatus[];

/**
 * The representative status for each step, index-aligned with getStepLabels.
 * Used to pick a per-step icon without duplicating the fulfillment split.
 */
export function getStepStatuses(
  fulfillment: Order['fulfillmentType']
): readonly OrderStatus[] {
  return isPickupFulfillment(fulfillment) ? PICKUP_STEP_STATUSES : DELIVERY_STEP_STATUSES;
}

/**
 * The active step index (0-based) for a status. Returns -1 for statuses with no
 * place on the bar (pending before confirm, cancelled, refunded).
 */
export function getStepIndex(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType']
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
      // The pivot: pickup advances to "Ready for pickup" (step 2); delivery stays
      // at "Preparing" (step 1) until a carrier picks it up.
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
 * A friendly status sentence for order cards and inline status rows. Pickup
 * wording never mentions a driver or delivery.
 */
export function getStatusLine(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType']
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

/** Short chip label (Title Case) for the status chip on the detail page. */
export function getChipLabel(
  status: OrderStatus,
  fulfillment: Order['fulfillmentType']
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

/** Section heading for the address block — a pickup order has no drop address. */
export function getAddressHeading(fulfillment: Order['fulfillmentType']): string {
  return isPickupFulfillment(fulfillment) ? 'Collect from' : 'Delivery Address';
}

/** The fee row label: pickup is collected, so it is never a "delivery fee". */
export function getFeeRowLabel(fulfillment: FulfillmentType): string {
  return isPickupFulfillment(fulfillment) ? 'Pickup' : 'Delivery fee';
}
