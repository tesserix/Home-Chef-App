import type { FulfillmentType, OrderStatus } from '@/shared/types';
import type { OrderPhotoKind } from '@/shared/services/upload-service';

/** Carrier the chef can hand a delivery order to, mid-flight. Sent as the
 *  optional `carrier` on PUT /chef/orders/:id/status. */
export type Carrier = 'chef_delivery' | 'delivery';

export interface CarrierSwitch {
  label: string;
  carrier: Carrier;
  /** Why this matters right now — surfaced when it is the ONLY way the order
   *  can still be completed. */
  hint?: string;
}

/**
 * The single status transition a chef can drive from a live order card, or the
 * caption to show when the order is out of their hands. This mirrors the vendor
 * app's `FooterActions` (apps/mobile-vendor/app/orders/[orderId].tsx) so a chef
 * sees the same steps — and the same photo requirements — on either surface.
 *
 * `photoKind` is the lifecycle photo that must be uploaded BEFORE the status
 * moves. Null means the transition needs no photo.
 */
export type ChefOrderAction =
  | {
      kind: 'advance';
      label: string;
      nextStatus: OrderStatus;
      photoKind: OrderPhotoKind | null;
      variant: 'default' | 'success';
      /** Carrier committed alongside the transition (Mark Ready only). */
      carrier?: Carrier;
      switchTo?: CarrierSwitch;
    }
  | { kind: 'waiting'; caption: string; switchTo?: CarrierSwitch }
  | null;

/** Chef/platform carrier capabilities, returned per order by /chef/orders. */
export interface CarrierCapabilities {
  /** The chef's "I deliver myself" toggle. */
  offersSelfDelivery?: boolean;
  /** Whether any 3PL provider is enabled — i.e. whether a rider can be
   *  dispatched at all. False means nobody is coming for a `delivery` order. */
  riderDispatchAvailable?: boolean;
}

const SELF_DELIVER_SWITCH: CarrierSwitch = {
  label: "I'll deliver this instead",
  carrier: 'chef_delivery',
};

const RIDER_SWITCH: CarrierSwitch = {
  label: 'Hand to a rider instead',
  carrier: 'delivery',
};

/**
 * Resolves the chef's next step for a live order.
 *
 * Photo-gated steps (matching the vendor app):
 *  - preparing → ready needs the food-ready photo (the customer sees it).
 *  - ready → delivered on a PICKUP order needs the proof-of-handover photo
 *    (it is the evidence in a "I never collected it" dispute).
 *
 * The carrier escape matters as much as the steps themselves: a `delivery`
 * order at `ready` is waiting on a rider, and if no 3PL provider is enabled NO
 * rider is ever dispatched. Without an "I'll deliver this instead" switch such
 * an order can never reach `delivered` — it just sits in the live queue forever.
 */
export function getChefOrderAction(
  status: OrderStatus,
  fulfillmentType: FulfillmentType = 'delivery',
  capabilities: CarrierCapabilities = {}
): ChefOrderAction {
  const { offersSelfDelivery = false, riderDispatchAvailable = false } = capabilities;

  switch (status) {
    case 'accepted':
      return {
        kind: 'advance',
        label: 'Start Preparing',
        nextStatus: 'preparing',
        photoKind: null,
        variant: 'default',
      };
    case 'preparing': {
      // A delivery order marked ready with no rider to dispatch to would land
      // straight in the dead-end below. When the chef self-delivers and 3PL is
      // dark, commit the carrier here so the order goes out on a route that can
      // actually finish. With riders live, leave it on the 3PL path as before.
      const commitSelfDelivery =
        fulfillmentType === 'delivery' && offersSelfDelivery && !riderDispatchAvailable;
      return {
        kind: 'advance',
        label:
          fulfillmentType === 'pickup'
            ? 'Mark ready for pickup'
            : commitSelfDelivery
              ? "Mark Ready · I'll deliver"
              : 'Mark Ready',
        nextStatus: 'ready',
        photoKind: 'ready',
        variant: 'success',
        ...(commitSelfDelivery ? { carrier: 'chef_delivery' as Carrier } : {}),
      };
    }
    case 'ready':
      // Pickup: the customer collects, so the chef closes the order out. The API
      // rejects a carrier on pickup orders — there is nothing to hand off.
      if (fulfillmentType === 'pickup') {
        return {
          kind: 'advance',
          label: 'Mark handed over',
          nextStatus: 'delivered',
          photoKind: 'handover',
          variant: 'success',
        };
      }
      // The chef took it on themselves — they walk it out, and can still hand it
      // back to a rider while it hasn't left the kitchen.
      if (fulfillmentType === 'chef_delivery') {
        return {
          kind: 'advance',
          label: 'Out for delivery',
          nextStatus: 'picked_up',
          photoKind: null,
          variant: 'default',
          ...(riderDispatchAvailable ? { switchTo: RIDER_SWITCH } : {}),
        };
      }
      // 3PL delivery. Nobody is coming when rider dispatch is off, so say that
      // plainly rather than showing an indefinite "waiting" the chef can't act on.
      return {
        kind: 'waiting',
        caption: riderDispatchAvailable
          ? 'Waiting for a rider to pick up'
          : 'No delivery partner is available',
        ...(offersSelfDelivery
          ? {
              switchTo: {
                ...SELF_DELIVER_SWITCH,
                ...(riderDispatchAvailable
                  ? {}
                  : { hint: 'Deliver it yourself to complete this order.' }),
              },
            }
          : {}),
      };
    case 'picked_up':
      // Only a self-delivering chef completes the order; a 3PL rider owns the
      // last mile on a plain `delivery` order.
      if (fulfillmentType === 'chef_delivery') {
        return {
          kind: 'advance',
          label: 'Mark delivered',
          nextStatus: 'delivered',
          photoKind: null,
          variant: 'success',
        };
      }
      return { kind: 'waiting', caption: 'Out for delivery' };
    default:
      return null;
  }
}

/**
 * Whether an order still belongs in the chef's LIVE queue. Mirrors the vendor
 * app's `isHistoryOrder` (apps/mobile-vendor/app/(tabs)/orders.tsx), inverted.
 *
 * A `chef_delivery` order at `picked_up` means the CHEF is out delivering it —
 * still their job, and still needing a "Mark delivered" tap — so it stays live.
 * A 3PL/pickup order at `picked_up` is out of their hands, and so is any order
 * under an open delivery-failure review (an admin decides the money).
 */
export function isLiveChefOrder(order: {
  status: OrderStatus;
  fulfillmentType?: FulfillmentType;
  deliveryFailureReported?: boolean;
}): boolean {
  if (order.deliveryFailureReported) return false;
  if (order.status === 'picked_up') return order.fulfillmentType === 'chef_delivery';
  return true;
}

/** Caption shown under a photo-gated button so the requirement is named before
 *  the chef clicks, not after the file dialog surprises them. */
export function photoRequirementCaption(photoKind: OrderPhotoKind): string {
  return photoKind === 'ready'
    ? "You'll attach a photo of the prepared order"
    : "You'll attach a photo of the handover";
}
