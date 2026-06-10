import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useRouter } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';

interface OrderCardProps {
  order: Order;
}

// Status chip: tint bg + dark text of same family per spec §2.7 / §2 item 7.
// pending/confirmed/preparing/ready → coral-tint bg + coral-pressed text (active)
// picked_up/delivered → success-tint bg + success text
// cancelled → surface-soft bg + charcoal-soft text (neutral)
type ChipStyle = { bg: string; text: string; label: string };

function getStatusChip(status: Order['status']): ChipStyle {
  switch (status) {
    case 'pending':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Pending',
      };
    case 'confirmed':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Confirmed',
      };
    case 'preparing':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Preparing',
      };
    case 'ready':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Ready',
      };
    case 'picked_up':
      return {
        bg: customerColors.success.tint,
        text: customerColors.success.DEFAULT,
        label: 'Picked Up',
      };
    case 'delivered':
      return {
        bg: customerColors.success.tint,
        text: customerColors.success.DEFAULT,
        label: 'Delivered',
      };
    case 'cancelled':
      return {
        bg: customerColors.surface.soft,
        text: customerColors.charcoal.soft,
        label: 'Cancelled',
      };
    default:
      return {
        bg: customerColors.surface.soft,
        text: customerColors.charcoal.soft,
        label: status,
      };
  }
}

// Tabular date for order row (e.g. "12 Jun 2026")
function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

export function OrderCard({ order }: OrderCardProps) {
  const router = useRouter();
  const chip = getStatusChip(order.status);
  const itemCount = order.items.reduce((sum, item) => sum + item.quantity, 0);

  function handlePress() {
    router.push(`/order/${order.id}`);
  }

  // iOS Pressable bug: visual styles live on an inner View; the Pressable is
  // a plain wrapper. flex-row wrapper fills the row, opacity on press handled
  // by the Pressable's android_ripple / opacity prop.
  return (
    <Pressable
      onPress={handlePress}
      accessibilityRole="button"
      accessibilityLabel={`Order #${order.orderNumber} from ${order.chef.name}`}
      style={styles.pressableWrapper}
    >
      {({ pressed }) => (
        // Shadow must be on the outer clip view; overflow hidden on same view
        // kills shadow on iOS so we split: shadow on the outer View only,
        // clip radius on the inner content View.
        <View style={[styles.cardOuter, pressed && styles.cardPressed]}>
          <View style={styles.cardInner}>
            {/* Top row: chef name + status chip */}
            <View style={styles.topRow}>
              <View style={styles.chefInfo}>
                <Text
                  style={styles.chefName}
                  numberOfLines={1}
                  ellipsizeMode="tail"
                >
                  {order.chef.name}
                </Text>
                <Text style={styles.orderNumber}>#{order.orderNumber}</Text>
              </View>

              {/* Status chip — radius-full, tint bg + dark text */}
              <View style={[styles.statusChip, { backgroundColor: chip.bg }]}>
                <Text style={[styles.statusText, { color: chip.text }]}>
                  {chip.label}
                </Text>
              </View>
            </View>

            {/* Hairline separator */}
            <View style={styles.hairline} />

            {/* Bottom row: item count + date (charcoal-soft, tabular) + total (charcoal, tabular) */}
            <View style={styles.bottomRow}>
              <Text style={styles.meta}>
                {itemCount} {itemCount === 1 ? 'item' : 'items'}
                {'  ·  '}
                {formatDate(order.createdAt)}
              </Text>
              <Text style={styles.total}>₹{order.totalAmount.toFixed(0)}</Text>
            </View>
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  pressableWrapper: {
    // flex-1 wrapper so it fills whatever column/list cell the parent gives
    marginHorizontal: 16,
    marginVertical: 5,
  },

  // Shadow lives here (NOT on cardInner) — iOS kills shadow if overflow+radius
  // are on the same View as shadowColor/shadowOffset.
  cardOuter: {
    borderRadius: 12,
    backgroundColor: customerColors.canvas,
    // Hairline border via shadowColor trick — use borderColor instead for clarity
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    // Soft shadow — shadow[1] style
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.04,
    shadowRadius: 3,
    elevation: 1,
  },
  cardPressed: {
    opacity: 0.92,
  },

  // Content clips to radius — overflow hidden here is safe because shadow
  // is on the parent cardOuter.
  cardInner: {
    borderRadius: 12,
    overflow: 'hidden',
    paddingHorizontal: 16,
    paddingVertical: 14,
  },

  // — Chef name + order number column
  topRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 10,
  },
  chefInfo: {
    flex: 1,
    gap: 3,
  },
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: 0,
  },
  orderNumber: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    // Tabular numerals so the hash-number aligns neatly
    fontVariant: ['tabular-nums'],
  },

  // — Status chip (radius-full, tint bg + dark text, Inter-SemiBold caption)
  statusChip: {
    alignSelf: 'flex-start',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 9999,
  },
  statusText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.1,
  },

  // — Hairline separator (not a heavy border)
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginVertical: 12,
  },

  // — Meta + total row
  bottomRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    // Tabular figures for date alignment
    fontVariant: ['tabular-nums'],
    flex: 1,
  },
  total: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    // Tabular numerals so price digits are monospaced
    fontVariant: ['tabular-nums'],
  },
});
