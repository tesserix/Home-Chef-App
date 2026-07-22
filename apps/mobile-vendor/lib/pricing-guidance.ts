/**
 * Price guidance for menu items.
 *
 * A home chef competes with restaurants on price for comparable food. Priced
 * at parity, the customer has no reason to choose the chef — they order from a
 * restaurant instead, and the chef gets no order at all. Chefs set prices in
 * isolation with no view of that, so the form nudges at the moment they type.
 *
 * This is guidance, never a block: the chef knows their dish, their portion
 * and their ingredient cost better than we do. A premium biryani for six can
 * legitimately sit above the line.
 */

/**
 * The rupee point at which a single dish starts reading as restaurant pricing
 * to an Indian customer. Deliberately generous — the warning has to stay rare
 * enough that chefs read it rather than learn to dismiss it.
 */
export const RESTAURANT_PARITY_PRICE = 500;

export interface PricingHint {
  tone: 'warn';
  message: string;
}

/**
 * Returns guidance for the price as typed, or null when there is nothing
 * useful to say.
 *
 * Non-numeric, empty and non-positive values return null: those are the
 * validator's job, and doubling up would put two messages under one field.
 */
export function pricingHint(price: string): PricingHint | null {
  const value = Number(String(price).trim());
  if (!Number.isFinite(value) || value <= 0) return null;

  if (value >= RESTAURANT_PARITY_PRICE) {
    return {
      tone: 'warn',
      message:
        `At ₹${Math.round(value)} this is restaurant pricing. Customers pick home chefs ` +
        `for better food at a lower price — if it costs the same, they'll order from a ` +
        `restaurant instead. Keep it below ₹${RESTAURANT_PARITY_PRICE} unless the portion ` +
        `genuinely justifies it.`,
    };
  }
  return null;
}
