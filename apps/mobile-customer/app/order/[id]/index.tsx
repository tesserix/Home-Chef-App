import React from 'react';
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { ChevronLeft, MapPin, Navigation } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';
import { startOrderPayment } from '../../../lib/payment';
import { CookingIndicator } from '../../../components/status/CookingIndicator';
import { OrderProgressBar } from '../../../components/orders/OrderProgressBar';
import type { Order } from '../../../types/customer';

const ACTIVE_STATUSES: Order['status'][] = [
  'accepted',
  'preparing',
  'ready',
  'picked_up',
  'delivering',
];

// ─── Status meta ──────────────────────────────────────────────────────────────

type StatusChipStyle = { bg: string; text: string; label: string };

function getStatusChipStyle(status: Order['status']): StatusChipStyle {
  switch (status) {
    case 'pending':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Pending',
      };
    case 'accepted':
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
        label: 'Ready for Pickup',
      };
    case 'picked_up':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'On the Way',
      };
    case 'delivering':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'Out for Delivery',
      };
    case 'refunded':
      return {
        bg: customerColors.surface.soft,
        text: customerColors.charcoal.soft,
        label: 'Refunded',
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

function formatDateTime(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// ─── Item row ─────────────────────────────────────────────────────────────────

interface ItemRowProps {
  name: string;
  quantity: number;
  price: number;
  isLast: boolean;
}

function ItemRow({ name, quantity, price, isLast }: ItemRowProps) {
  return (
    <View style={[itemStyles.row, !isLast && itemStyles.rowBorder]}>
      {/* Quantity chip */}
      <View style={itemStyles.qtyChip}>
        <Text style={itemStyles.qtyText}>{quantity}</Text>
      </View>
      {/* Name */}
      <Text style={itemStyles.name} numberOfLines={2}>
        {name}
      </Text>
      {/* Line price */}
      <Text style={itemStyles.price}>
        ₹{(price * quantity).toFixed(0)}
      </Text>
    </View>
  );
}

const itemStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 12,
  },
  rowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  qtyChip: {
    width: 28,
    height: 28,
    borderRadius: 6,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  qtyText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  name: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 20,
  },
  price: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function OrderDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, isError } = useOrder(id ?? '');
  const [paying, setPaying] = React.useState(false);

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.centered}>
          <ActivityIndicator
            size="large"
            color={customerColors.coral.DEFAULT}
          />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !data?.data) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.centered}>
          <Text style={styles.errorText}>Order not found</Text>
          <Pressable
            onPress={() => router.back()}
            accessibilityRole="button"
            accessibilityLabel="Go back"
          >
            <View style={styles.ghostButton}>
              <Text style={styles.ghostButtonText}>Go Back</Text>
            </View>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  const order = data.data;
  const chipStyle = getStatusChipStyle(order.status);
  const isActiveOrder = ACTIVE_STATUSES.includes(order.status);
  const isPreparing = order.status === 'preparing';

  const subtotal = order.items.reduce(
    (sum, item) => sum + item.price * item.quantity,
    0,
  );
  const deliveryFee = order.totalAmount - subtotal;

  function handleTrackOrder() {
    router.push(`/order/${order.id}/track`);
  }

  const needsPayment =
    (order.paymentStatus === 'pending' || order.paymentStatus === 'failed') &&
    order.status !== 'cancelled' &&
    order.status !== 'delivered';

  async function handlePayNow() {
    setPaying(true);
    try {
      await startOrderPayment(order.id);
    } finally {
      setPaying(false);
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* ── Nav row ─────────────────────────────────────────────────── */}
        <View style={styles.navRow}>
          <Pressable
            onPress={() => router.back()}
            accessibilityRole="button"
            accessibilityLabel="Go back"
            hitSlop={8}
          >
            <View style={styles.backCircle}>
              <ChevronLeft
                size={20}
                color={customerColors.charcoal.DEFAULT}
                accessibilityElementsHidden
              />
            </View>
          </Pressable>
          <Text style={styles.navTitle} numberOfLines={1}>
            Order details
          </Text>
          {/* Spacer — mirrors back button width so the title is centred */}
          <View style={styles.navSpacer} />
        </View>

        {/* ── Hero status section ─────────────────────────────────────── */}
        <View style={styles.heroSection}>
          {/* Chef + order reference */}
          <Text style={styles.heroChefName} numberOfLines={1}>
            {order.chef?.name ?? 'Your order'}
          </Text>
          <Text style={styles.heroOrderRef}>
            Order #{order.orderNumber} · {formatDateTime(order.createdAt)}
          </Text>

          {/* Status chip row — cooking animation when preparing */}
          <View style={styles.chipRow}>
            {isPreparing ? (
              <CookingIndicator
                size={18}
                color={customerColors.coral.DEFAULT}
                style={styles.cookingIcon}
              />
            ) : null}
            <View style={[styles.statusChip, { backgroundColor: chipStyle.bg }]}>
              <Text style={[styles.statusChipText, { color: chipStyle.text }]}>
                {isPreparing ? 'Cooking now' : chipStyle.label}
              </Text>
            </View>
            {order.estimatedDeliveryTime ? (
              <Text style={styles.etaText}>
                ETA {order.estimatedDeliveryTime}
              </Text>
            ) : null}
          </View>

          {/* Progress bar — only for active orders */}
          {isActiveOrder || order.status === 'pending' ? (
            <View style={styles.progressWrapper}>
              <OrderProgressBar status={order.status} />
            </View>
          ) : null}

          {/* Payment pending warning */}
          {needsPayment ? (
            <View style={styles.paymentWarning}>
              <Text style={styles.paymentWarningText}>
                Payment pending — complete to confirm your order.
              </Text>
            </View>
          ) : null}
        </View>

        {/* ── CTAs ──────────────────────────────────────────────────────── */}
        {needsPayment && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={handlePayNow}
              disabled={paying}
              accessibilityRole="button"
              accessibilityLabel="Pay now for this order"
            >
              <View style={styles.primaryButton}>
                {paying ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.primaryButtonText}>Pay now</Text>
                )}
              </View>
            </Pressable>
          </View>
        )}

        {isActiveOrder && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={handleTrackOrder}
              accessibilityRole="button"
              accessibilityLabel="Track this order"
            >
              <View style={styles.primaryButton}>
                <Navigation
                  size={16}
                  color={customerColors.canvas}
                  accessibilityElementsHidden
                />
                <Text style={styles.primaryButtonText}>Track Order</Text>
              </View>
            </Pressable>
          </View>
        )}

        {order.status === 'delivered' && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={() => router.push(`/order/${order.id}/review`)}
              accessibilityRole="button"
              accessibilityLabel="Leave a review for this order"
            >
              <View style={styles.primaryButton}>
                <Text style={styles.primaryButtonText}>Leave a Review</Text>
              </View>
            </Pressable>
            <Pressable
              onPress={() => router.push(`/order/${order.id}/tip`)}
              accessibilityRole="button"
              accessibilityLabel="Tip your chef or rider"
              style={styles.secondaryPressable}
            >
              <View style={styles.outlineButton}>
                <Text style={styles.outlineButtonText}>Tip your chef / rider</Text>
              </View>
            </Pressable>
          </View>
        )}

        {/* ── Section: Items ────────────────────────────────────────────── */}
        <View style={styles.sectionDivider} />
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Items</Text>
          {order.items.map((item, index) => (
            <ItemRow
              key={String(index)}
              name={item.name}
              quantity={item.quantity}
              price={item.price}
              isLast={index === order.items.length - 1}
            />
          ))}
        </View>

        {/* ── Section: Price breakdown ──────────────────────────────────── */}
        <View style={styles.sectionDivider} />
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Price breakdown</Text>
          {/* Soft card wraps the breakdown rows */}
          <View style={styles.priceCard}>
            <View style={styles.priceRow}>
              <Text style={styles.priceLabel}>Subtotal</Text>
              <Text style={styles.priceValue}>₹{subtotal.toFixed(2)}</Text>
            </View>
            <View style={[styles.priceRow, styles.priceRowBorder]}>
              <Text style={styles.priceLabel}>Delivery fee</Text>
              <Text style={styles.priceValue}>₹{deliveryFee.toFixed(2)}</Text>
            </View>
            {/* Total — heavier weight + hairline above */}
            <View style={[styles.priceRow, styles.totalRow]}>
              <Text style={styles.totalLabel}>Total</Text>
              <Text style={styles.totalValue}>
                ₹{order.totalAmount.toFixed(2)}
              </Text>
            </View>
          </View>
        </View>

        {/* ── Section: Delivery address ─────────────────────────────────── */}
        <View style={styles.sectionDivider} />
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Delivery address</Text>
          <View style={styles.addressCard}>
            <View style={styles.addressIconCol}>
              <MapPin
                size={16}
                color={customerColors.coral.DEFAULT}
                accessibilityElementsHidden
              />
            </View>
            <View style={styles.addressTextCol}>
              <Text style={styles.addressLine}>
                {order.deliveryAddress.addressLine1}
              </Text>
              {order.deliveryAddress.addressLine2 ? (
                <Text style={styles.addressLine}>
                  {order.deliveryAddress.addressLine2}
                </Text>
              ) : null}
              <Text style={styles.addressLine}>
                {order.deliveryAddress.city},{' '}
                {order.deliveryAddress.state}{' '}
                {order.deliveryAddress.pincode}
              </Text>
            </View>
          </View>
        </View>

        {/* Bottom breathing room */}
        <View style={styles.bottomPad} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 16,
  },
  errorText: {
    fontFamily: 'Inter',
    fontSize: 16,
    color: customerColors.charcoal.soft,
  },
  ghostButton: {
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ghostButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },

  scrollContent: {
    paddingBottom: 32,
  },

  // ── Navigation row ────────────────────────────────────────────────────────
  navRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 8,
    gap: 8,
  },
  backCircle: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  navTitle: {
    flex: 1,
    textAlign: 'center',
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.1,
  },
  // Same width as the back circle so the title is optically centered.
  navSpacer: {
    width: 36,
  },

  // ── Hero status section ───────────────────────────────────────────────────
  heroSection: {
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  heroChefName: {
    fontFamily: 'Geist-SemiBold',
    fontSize: 22,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.3,
    marginBottom: 4,
  },
  heroOrderRef: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
    marginBottom: 14,
  },

  // Chip + ETA row
  chipRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    flexWrap: 'wrap',
  },
  cookingIcon: {
    marginRight: -2,
  },
  statusChip: {
    alignSelf: 'flex-start',
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 9999,
  },
  statusChipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    letterSpacing: 0.1,
  },
  etaText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },

  // Progress bar wrapper — full-bleed (no horizontal padding, it handles its own)
  progressWrapper: {
    marginTop: 16,
    marginHorizontal: -16,
  },

  // Payment warning banner
  paymentWarning: {
    marginTop: 12,
    backgroundColor: customerColors.coral.tint,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  paymentWarningText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.coral.pressed,
    lineHeight: 18,
  },

  // ── CTAs ──────────────────────────────────────────────────────────────────
  ctaWrapper: {
    paddingHorizontal: 16,
    paddingTop: 12,
    gap: 10,
  },
  // Coral primary button — iOS Pressable gotcha: styles on inner View.
  primaryButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    minHeight: 52,
    paddingHorizontal: 24,
  },
  primaryButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.canvas,
  },
  // Coral outline secondary button
  secondaryPressable: {
    marginTop: 2,
  },
  outlineButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.canvas,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
    minHeight: 52,
    paddingHorizontal: 24,
  },
  outlineButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.coral.DEFAULT,
  },

  // ── Section skeleton ──────────────────────────────────────────────────────
  sectionDivider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginHorizontal: 16,
    marginTop: 12,
  },
  section: {
    paddingHorizontal: 16,
    paddingVertical: 16,
  },
  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 12,
    letterSpacing: -0.1,
  },

  // ── Price breakdown card ──────────────────────────────────────────────────
  priceCard: {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 4,
  },
  priceRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 12,
  },
  priceRowBorder: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  priceLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },
  priceValue: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  totalRow: {
    borderTopWidth: 1,
    borderTopColor: customerColors.hairline,
    marginTop: 0,
  },
  totalLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
  },
  totalValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  // ── Delivery address card ─────────────────────────────────────────────────
  addressCard: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 12,
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  addressIconCol: {
    marginTop: 2,
  },
  addressTextCol: {
    flex: 1,
  },
  addressLine: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 22,
  },

  bottomPad: {
    height: 16,
  },
});
