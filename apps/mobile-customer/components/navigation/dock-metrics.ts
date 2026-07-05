// dock-metrics — shared geometry for the floating dock and the layers that
// anchor to it (CartFab, screen scroll padding, Home's active-order card).
// Lives in its own module so Dock and CartFab can both import it without a
// circular dependency (Dock → CartFab → Dock left DOCK_HEIGHT undefined at
// style-creation time, which dropped the FAB onto the bar).

import { useSafeAreaInsets } from 'react-native-safe-area-context';

/** Dock bar height. */
export const DOCK_HEIGHT = 64;

/** Gap between the dock and the bottom safe-area inset. Kept tight so the
    dock hugs the home indicator (owner feedback: closer to the bottom). */
export const DOCK_BOTTOM_GAP = 4;

/**
 * Bottom padding a screen needs so scrollable content clears the floating
 * dock (dock height + gap + safe-area + a breathing row of space).
 */
export function useDockClearance(): number {
  const insets = useSafeAreaInsets();
  return insets.bottom + DOCK_BOTTOM_GAP + DOCK_HEIGHT + 12;
}
