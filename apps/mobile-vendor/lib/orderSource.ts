import { theme } from '@homechef/mobile-shared/theme';

import type { OrderSource } from '../hooks/useVendorOrders';

// Presentation for the order-source badge on the vendor feed (#435). à-la-carte
// is the default and gets NO badge (label null) to keep the common case clean —
// only meal-plan / subscription / group orders are called out.

export interface OrderSourceBadge {
  label: string;
  /** Text color — near-black ink for a WCAG-AA contrast floor on the tint bg. */
  color: string;
  /** Tinted background that carries the category hue. */
  bg: string;
}

/** The short label for a source, or null for à-la-carte / unknown (no badge). */
export function orderSourceLabel(source?: OrderSource | string): string | null {
  switch (source) {
    case 'meal_plan':
      return 'Meal plan';
    case 'subscription':
      return 'Subscription';
    case 'group':
      return 'Group';
    default:
      return null;
  }
}

/** The badge (label + colors) for a source, or null when no badge should show.
 *  The hue lives in the tinted background; the text is near-black ink so the
 *  11px label clears the WCAG AA 4.5:1 floor (the amber/info DEFAULT shades on
 *  their own tints do not — .impeccable.md mandates AA). */
export function orderSourceBadge(source?: OrderSource | string): OrderSourceBadge | null {
  const label = orderSourceLabel(source);
  if (!label) return null;
  const tint =
    source === 'meal_plan'
      ? theme.colors.info.tint
      : source === 'subscription'
        ? theme.colors.amber.tint
        : theme.colors.paprika.tint; // group
  return { label, color: theme.colors.ink.DEFAULT, bg: tint };
}
