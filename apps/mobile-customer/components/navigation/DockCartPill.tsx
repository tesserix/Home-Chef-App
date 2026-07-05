// DockCartPill — the dock's dynamic fifth slot. Renders nothing while the
// cart is empty; when items exist it appears at the dock's right end after a
// thin divider as a solid-coral pill (bag icon + running total, item count as
// a mini badge on the bag) and routes to checkout. Replaces the old
// full-width CartBar that docked flush on the legacy tab bar.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import { ShoppingBag } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useCartStore } from '../../store/cart-store';

const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

export function DockCartPill() {
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
      style={styles.wrapper}
    >
      <View style={styles.divider} />
      <Pressable
        onPress={() => router.push('/checkout')}
        accessibilityRole="button"
        accessibilityLabel={`View cart — ${itemCount} ${
          itemCount === 1 ? 'item' : 'items'
        }, ₹${total.toFixed(2)}`}
      >
        <View style={styles.pill}>
          <View>
            <ShoppingBag size={18} color={customerColors.canvas} />
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{itemCount}</Text>
            </View>
          </View>
          <Text style={styles.totalText}>₹{Math.round(total)}</Text>
        </View>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    paddingRight: 2,
  },
  divider: {
    width: StyleSheet.hairlineWidth,
    height: 28,
    backgroundColor: customerColors.hairline,
  },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    minHeight: 44,
    paddingHorizontal: 14,
    borderRadius: 20,
    backgroundColor: customerColors.coral.DEFAULT,
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
