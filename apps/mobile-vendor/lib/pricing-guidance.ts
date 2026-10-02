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

/**
 * The same point for a bakery item, which is priced as a whole cake rather than
 * a single serving — a two-kilo truffle cake at ₹1400 is the going rate, not an
 * outlier.
 */
export const BAKERY_PARITY_PRICE = 2500;

export interface PricingHint {
  tone: 'warn';
  message: string;
}

export interface PricingHintOptions {
  /** True for a cake or bake, which is priced per whole item. */
  isBakery?: boolean;
  /** The parity lines are rupee figures, so only INR kitchens get a hint. */
  currency?: string | null;
}

/**
 * Returns guidance for the price as typed, or null when there is nothing
 * useful to say.
 *
 * Non-numeric, empty and non-positive values return null: those are the
 * validator's job, and doubling up would put two messages under one field.
 */
export function pricingHint(
  price: string,
  { isBakery = false, currency }: PricingHintOptions = {},
): PricingHint | null {
  if ((currency ?? 'INR').toUpperCase() !== 'INR') return null;
  const value = Number(String(price).trim());
  if (!Number.isFinite(value) || value <= 0) return null;

  if (isBakery) {
    if (value < BAKERY_PARITY_PRICE) return null;
    return {
      tone: 'warn',
      message:
        `At ₹${Math.round(value)} this is above what a bakery charges for a cake this ` +
        `size. Customers order from a home baker for a better cake at a fairer price — ` +
        `keep it below ₹${BAKERY_PARITY_PRICE} unless the tiers, weight or decoration ` +
        `genuinely justify it.`,
    };
  }

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
