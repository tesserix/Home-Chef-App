/**
 * Vendor-app DietIcon shim.
 *
 * The canonical implementation lives in `@homechef/mobile-shared/ui`.
 * This file re-exports it and adds a backwards-compat `isVeg: boolean`
 * prop alias so existing callers (MenuItemRow, MenuItemForm) don't need
 * to be updated.
 */
import { DietIcon as SharedDietIcon } from '@homechef/mobile-shared/ui';

interface DietIconProps {
  /** Modern prop: explicit kind. Takes precedence over `isVeg`. */
  kind?: 'veg' | 'non-veg' | 'unknown';
  /** Legacy boolean alias. Ignored when `kind` is provided. */
  isVeg?: boolean;
  size?: number;
}

export function DietIcon({ kind, isVeg, size = 14 }: DietIconProps) {
  let resolvedKind: 'veg' | 'non-veg' | 'unknown';
  if (kind !== undefined) {
    resolvedKind = kind;
  } else if (isVeg !== undefined) {
    resolvedKind = isVeg ? 'veg' : 'non-veg';
  } else {
    resolvedKind = 'unknown';
  }
  return <SharedDietIcon kind={resolvedKind} size={size} />;
}
