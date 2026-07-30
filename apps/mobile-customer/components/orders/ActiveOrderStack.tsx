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
import { ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { ActiveOrderCard } from './ActiveOrderCard';

// Android ripple tint — translucent token, never a new literal colour.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

interface ActiveOrderStackProps {
  orders: Order[];
}

export function ActiveOrderStack({ orders }: ActiveOrderStackProps) {
  const router = useRouter();

  if (orders.length === 0) return null;

  const [primary] = orders;
  const extra = orders.length - 1;

  return (
    <View>
      <ActiveOrderCard order={primary} />
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
    borderWidth: StyleSheet.hairlineWidth,
    borderTopWidth: 0,
    borderColor: customerColors.hairline,
  },
  moreText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
});
