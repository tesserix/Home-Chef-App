// CartFab — floating cart button hovering above the dock's right end (owner
// call: cart ON TOP of the tabs, not inside them — keeps the dock pure
// navigation with one accent, and the cart reads as an action, not a tab).
// Renders nothing while the cart is empty; with items it shows a solid-coral
// pill (bag + count badge + running total) and routes to checkout.
//
// Rendered by Dock inside its absolutely-positioned root, so it exists on
// every tab. Home lifts its active-order card when the cart is non-empty so
// the two floating layers never overlap (see app/(tabs)/index.tsx).

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { formatMoney } from '../../lib/format';
import { router } from 'expo-router';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import { ShoppingBag } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useCartStore } from '../../store/cart-store';
import { DOCK_HEIGHT } from './dock-metrics';

const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

/** Vertical gap between the dock top edge and the cart FAB. */
const FAB_GAP = 10;
/** FAB pill height — used by Home to lift the active-order card clear. */
export const CART_FAB_CLEARANCE = 44 + FAB_GAP + 6;

export function CartFab() {
  const items = useCartStore((s) => s.items);
  const total = useCartStore((s) => s.total());
  const reduceMotion = useReducedMotion();

  if (items.length === 0) {
    return null;
  }

  const itemCount = items.reduce((sum, i) => sum + i.quantity, 0);

  return (
    <Animated.View
      entering={
        reduceMotion ? undefined : FadeIn.duration(250).easing(ENTRANCE_EASING)
      }
      style={styles.anchor}
      pointerEvents="box-none"
    >
      <Pressable
        onPress={() => router.push('/checkout')}
        accessibilityRole="button"
        accessibilityLabel={`View cart — ${itemCount} ${
          itemCount === 1 ? 'item' : 'items'
        }, ${formatMoney(total)}`}
      >
        <View style={styles.pill}>
          <View>
            <ShoppingBag size={18} color={customerColors.canvas} />
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{itemCount}</Text>
            </View>
          </View>
          <Text style={styles.totalText}>{formatMoney(total)}</Text>
        </View>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  // Anchored above the dock bar's right end (Dock's root is the positioning
  // context: left/right 16 at bottom inset + gap).
  anchor: {
    position: 'absolute',
    right: 0,
    bottom: DOCK_HEIGHT + FAB_GAP,
  },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    height: 44,
    paddingHorizontal: 16,
    borderRadius: 22,
    backgroundColor: customerColors.coral.DEFAULT,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.14,
    shadowRadius: 12,
    elevation: 10,
  },
  badge: {
    position: 'absolute',
    top: -5,
    right: -7,
    minWidth: 15,
    height: 15,
    borderRadius: 8,
    paddingHorizontal: 3,
    backgroundColor: customerColors.canvas,
    alignItems: 'center',
    justifyContent: 'center',
  },
  badgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 9,
    color: customerColors.coral.DEFAULT,
  },
  totalText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.canvas,
    fontVariant: ['tabular-nums'],
  },
});
