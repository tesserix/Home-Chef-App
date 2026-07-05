import { theme } from '@homechef/mobile-shared/theme';

import type { OrderSource } from '../hooks/useVendorOrders';

// Presentation for the order-source badge on the vendor feed (#435). à-la-carte
// is the default and gets NO badge (label null) to keep the common case clean —
// only meal-plan / subscription / group orders are called out.

export interface OrderSourceBadge {
  label: string;
  color: string;
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

/** The badge (label + colors) for a source, or null when no badge should show. */
export function orderSourceBadge(source?: OrderSource | string): OrderSourceBadge | null {
  const label = orderSourceLabel(source);
  if (!label) return null;
  const palette =
    source === 'meal_plan'
      ? theme.colors.info
      : source === 'subscription'
        ? theme.colors.amber
        : theme.colors.paprika; // group
  return { label, color: palette.DEFAULT, bg: palette.tint };
}
