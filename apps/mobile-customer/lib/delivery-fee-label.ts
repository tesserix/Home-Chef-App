// The delivery claim on a chef card and a chef header, in one place so the two
// can never say different things about the same kitchen.
//
// A card cannot state the fee: it is distance-based and the customer's address
// isn't known until checkout. It CAN state a floor the quote only rises from —
// which is why the API sends the fee at zero distance plus whether that floor is
// also the ceiling. Saying "Free delivery" off a fee the API hardcoded to zero,
// while the order charged 39.12, is D-01.

export function deliveryFeeLabel(
  fee: number | undefined,
  flat: boolean | undefined,
): string | undefined {
  if (fee == null) return undefined;
  const free = fee < 0.005;
  // A flat fee holds however far away the customer is, so it can be stated plainly.
  if (flat) return free ? 'Free delivery' : `₹${fee} delivery`;
  // Otherwise distance can lift it, and the claim has to say so.
  return free ? 'Free delivery nearby' : `Delivery from ₹${fee}`;
}
