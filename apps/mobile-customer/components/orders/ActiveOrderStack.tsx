// Floating active-order tracker for the Home screen.
//
// ONE card — the most recent in-flight order — plus a quiet "N more active
// orders" row when there are others. Tapping that row opens the Orders tab,
// which is the surface built to list them.
//
// This replaced a collapsible card stack (front card + peeking edges, tap to
// expand the full list in place). With two orders it read as clutter: the peek
// edges sat over the chef photography, the overlay grew with the order count,
// and expanding shifted the feed underneath. A single card keeps the home screen
// photo-forward and chrome-light, and it costs one tap to see the rest —
// unbounded by how many orders are in flight.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useRouter } from 'expo-router';
import { ChevronRight, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { ActiveOrderCard } from './ActiveOrderCard';
import { HAIRLINE } from '../../lib/hairline';

// Android ripple tint — translucent token, never a new literal colour.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

interface ActiveOrderStackProps {
  orders: Order[];
  // Dismisses the card on screen. Omitted → no hide affordance is drawn.
  onHide?: (order: Order) => void;
}

export function ActiveOrderStack({ orders, onHide }: ActiveOrderStackProps) {
  const router = useRouter();

  if (orders.length === 0) return null;

  const [primary] = orders;
  const extra = orders.length - 1;

  return (
    <View>
      <ActiveOrderCard order={primary} />
      {onHide && primary ? (
        <Pressable
          onPress={() => onHide(primary)}
          style={styles.hideHit}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Hide this order card. It comes back when your order moves on."
        >
          {({ pressed }) => (
            <View style={[styles.hideDot, pressed && styles.hideDotPressed]}>
              <X size={12} color={customerColors.charcoal.soft} strokeWidth={2.5} />
            </View>
          )}
        </Pressable>
      ) : null}
      {extra > 0 ? (
        <Pressable
          onPress={() => router.navigate('/orders')}
          accessibilityRole="button"
          accessibilityLabel={`${extra} more active ${extra === 1 ? 'order' : 'orders'}. Opens your orders.`}
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          <View style={styles.moreRow}>
            <Text style={styles.moreText}>
              {extra} more active {extra === 1 ? 'order' : 'orders'}
            </Text>
            <ChevronRight size={16} color={customerColors.charcoal.soft} />
          </View>
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  // Straddles the card's top-right corner. A 44pt hit area (§spatial) around a
  // 22pt dot, so the control stays tappable without putting more chrome on a
  // card whose top row is already carrying the price badge and chevron.
  hideHit: {
    position: 'absolute',
    top: -10,
    right: 6,
    width: 44,
    height: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  hideDot: {
    width: 22,
    height: 22,
    borderRadius: 11,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 3,
    elevation: 2,
  },
  hideDotPressed: {
    opacity: 0.94,
  },

  // Sits directly under the card as one visual unit: square off the shared edge
  // so the two read as a single surface rather than two stacked cards — the
  // thing this component exists to stop doing.
  moreRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 44, // customer touch target (§spatial)
    marginTop: -8,
    paddingTop: 8,
    paddingHorizontal: 16,
    paddingBottom: 10,
    borderBottomLeftRadius: 16,
    borderBottomRightRadius: 16,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: HAIRLINE,
    borderTopWidth: 0,
    borderColor: customerColors.hairline,
  },
  moreText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
});
