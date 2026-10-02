import { formatMoney } from './format';

// The delivery claim on a chef card and a chef header, in one place so the two
// can never say different things about the same kitchen.
//
// Two ways to get this wrong, and we have made both:
//
//   1. Quoting a fee the order won't honour. A card cannot state the fee — it is
//      distance-based and the address isn't known until checkout — so the API
//      sends the fee at zero distance plus whether that floor is also the
//      ceiling. Saying "Free delivery" off a hardcoded zero, while the order
//      charged 39.12, was D-01.
//
//   2. Quoting a fee for a kitchen that cannot deliver at all. Delivery is
//      fulfillable only when the chef self-delivers or a 3PL is live, and every
//      provider is currently disabled — so a pickup-only kitchen was being
//      advertised at "Delivery from 39". `offersDelivery` is the server's own
//      answer to that question; a card that ignores it is quoting a price for a
//      service the customer cannot buy.

export function deliveryFeeLabel(
  fee: number | undefined,
  flat: boolean | undefined,
  offersDelivery: boolean | undefined,
  currency = 'INR',
): string | undefined {
  // Only `false` suppresses. Older API responses omit the field, and a missing
  // answer must not turn every kitchen into a pickup-only one.
  if (offersDelivery === false) return 'Pickup only';
  if (fee == null) return undefined;
  const free = fee < 0.005;
  // A flat fee holds however far away the customer is, so it can be stated plainly.
  if (flat) return free ? 'Free delivery' : `${formatMoney(fee, currency)} delivery`;
  // Otherwise distance can lift it, and the claim has to say so.
  return free ? 'Free delivery nearby' : `Delivery from ${formatMoney(fee, currency)}`;
}
