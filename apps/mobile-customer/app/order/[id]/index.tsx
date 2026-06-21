import React from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter, useIsFocused } from 'expo-router';
import { ChevronLeft, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';
import { useReorder } from '../../../hooks/useReorder';
import { useCartStore, makeLineId } from '../../../store/cart-store';
import { startOrderPayment } from '../../../lib/payment';
import { CookingIndicator } from '../../../components/status/CookingIndicator';
import { DeliveryMap } from '../../../components/tracking/DeliveryMap';
import { useOrderTracking } from '../../../hooks/useOrderTracking';
import { useOrderTrackingWS } from '../../../hooks/useOrderTrackingWS';
import type { Order } from '../../../types/customer';

const ACTIVE_STATUSES: Order['status'][] = [
  'accepted',
  'preparing',
  'ready',
  'picked_up',
  'delivering',
];

// Spec §2.7: status chip — tint bg + dark text of same family.
// in-progress  → coral-tint bg + coral-pressed text
// delivered    → success-tint bg + success text
// cancelled    → surface-soft bg + charcoal-soft text
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

// Human-readable inline status line for the map badge overlay
function getInlineStatusLabel(status: Order['status']): string {
  switch (status) {
    case 'accepted':
      return 'Order confirmed · chef is nearby';
    case 'preparing':
      return 'Chef is cooking your order nearby';
    case 'ready':
      return 'Ready for pickup · awaiting driver';
    case 'picked_up':
      return 'Your order is on the way';
    case 'delivering':
      return 'Out for delivery';
    default:
      return 'Tracking your order';
  }
}

export default function OrderDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const isFocused = useIsFocused();
  const { data, isLoading, isError } = useOrder(id ?? '');
  const [paying, setPaying] = React.useState(false);
  const reorder = useReorder();

  // Tracking hooks — mirror exact wiring from track.tsx.
  // Both hooks are always called (React hook rules); they short-circuit
  // internally when enabled=false so no network work happens for terminal orders.
  const orderId = id ?? '';
  const { driverLocation, isPollingFallback } = useOrderTrackingWS(
    orderId,
    isFocused,
  );
  const { data: trackingData } = useOrderTracking(orderId, isFocused);

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
          {/* Ghost button — white bg, hairline border, charcoal text */}
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

  // Derive effective driver coords — same logic as track.tsx.
  // Prefer real-time WS position; fall back to polling coords when WS failed.
  const tracking = trackingData?.data;
  const effectiveDriverLat =
    driverLocation != null && !isPollingFallback
      ? driverLocation.latitude
      : tracking?.delivery?.currentLatitude;
  const effectiveDriverLng =
    driverLocation != null && !isPollingFallback
      ? driverLocation.longitude
      : tracking?.delivery?.currentLongitude;

  const subtotal = order.items.reduce(
    (sum, item) => sum + item.price * item.quantity,
    0,
  );
  const deliveryFee = order.totalAmount - subtotal;

  // Reorder (#238) — fetch a re-validated preview, fill the cart with the
  // available lines (resolving current add-on option IDs), and route the
  // customer to checkout — or to the chef to review if anything changed.
  function handleReorder() {
    reorder.mutate(order.id, {
      onSuccess: (res) => {
        const available = res.items.filter((i) => i.available);
        if (available.length === 0) {
          Alert.alert('Unavailable', 'None of these items are available right now.');
          return;
        }

        const fillAndGo = () => {
          for (const it of available) {
            const modifiers = it.modifiers ?? [];
            useCartStore.getState().addItem(
              {
                lineId: makeLineId(it.menuItemId, modifiers),
                menuItemId: it.menuItemId,
                name: it.name,
                price: it.unitPrice,
                quantity: it.quantity,
                imageUrl: it.imageUrl,
                instructions: it.notes,
                modifiers: modifiers.length ? modifiers : undefined,
              },
              { id: res.chefId, name: res.chefName },
            );
          }
          const dropped = res.items.length - available.length;
          const needsReview = available.some((i) => i.needsReview);
          if (dropped > 0 || needsReview) {
            const msgs: string[] = [];
            if (dropped > 0) {
              msgs.push(`${dropped} item${dropped > 1 ? 's are' : ' is'} no longer available.`);
            }
            if (needsReview) msgs.push('Some add-ons changed — please review your cart.');
            Alert.alert('Review your cart', msgs.join(' '), [
              { text: 'OK', onPress: () => router.push(`/chef/${res.chefId}`) },
            ]);
          } else {
            router.push('/checkout');
          }
        };

        // Cross-chef conflict: confirm before replacing the current cart.
        const cart = useCartStore.getState();
        if (cart.chefId && cart.chefId !== res.chefId && cart.items.length > 0) {
          Alert.alert(
            'Replace cart?',
            'Your cart has items from another chef. Replace them with this order?',
            [
              { text: 'Cancel', style: 'cancel' },
              {
                text: 'Replace',
                style: 'destructive',
                onPress: () => {
                  useCartStore.getState().clearCart();
                  fillAndGo();
                },
              },
            ],
          );
        } else {
          fillAndGo();
        }
      },
      onError: () => Alert.alert('Error', 'Could not reorder right now. Please try again.'),
    });
  }

  // An order can be created but unpaid (verify failed, sheet dismissed, etc.).
  // Surface a clear Pay-now path so the customer can complete payment.
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
        {/* Back button row + header — flat white, hairline divider below */}
        <View style={styles.headerSection}>
          {/* Circular floating back button — charcoal chevron, no fill */}
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

          <View style={styles.headerText}>
            {/* Order API carries no chef object yet — fall back to a neutral title. */}
            <Text style={styles.chefName}>{order.chef?.name ?? 'Your order'}</Text>
            <Text style={styles.orderNumber}>Order #{order.orderNumber}</Text>
          </View>
        </View>

        {/* Status chip + ETA — spec §2.7 + §0 chip pattern */}
        <View style={styles.statusSection}>
          <View style={styles.statusChipRow}>
            {/* Live cooking animation while the chef is preparing the order (#50) */}
            {order.status === 'preparing' ? (
              <CookingIndicator size={20} color={customerColors.coral.DEFAULT} />
            ) : null}
            <View style={[styles.statusChip, { backgroundColor: chipStyle.bg }]}>
              <Text style={[styles.statusChipText, { color: chipStyle.text }]}>
                {order.status === 'preparing' ? 'Cooking now' : chipStyle.label}
              </Text>
            </View>
          </View>
          {order.estimatedDeliveryTime ? (
            <Text style={styles.etaText}>
              ETA: {order.estimatedDeliveryTime}
            </Text>
          ) : null}
          {needsPayment ? (
            <Text style={styles.paymentPendingText}>
              Payment pending — complete it to confirm your order.
            </Text>
          ) : null}
        </View>

        {/* Pay now — unpaid order recovery (verify failed / sheet dismissed). */}
        {needsPayment && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={handlePayNow}
              disabled={paying}
              accessibilityRole="button"
              accessibilityLabel="Pay now for this order"
            >
              <View style={styles.trackButton}>
                {paying ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.trackButtonText}>Pay now</Text>
                )}
              </View>
            </Pressable>
          </View>
        )}

        {/* Inline map card — replaces the "Track Order" button for active orders.
            Tapping the card navigates to the full-screen tracking experience.
            Hidden entirely for terminal statuses (delivered / cancelled / refunded). */}
        {isActiveOrder && (
          <View style={styles.mapCardWrapper}>
            {/* Pressable wraps the whole card — iOS Pressable inner-View pattern */}
            <Pressable
              onPress={() => router.push(`/order/${order.id}/track`)}
              accessibilityRole="button"
              accessibilityLabel="View live order tracking"
            >
              <View style={styles.mapCard}>
                {/* Map area — fixed height container so DeliveryMap's absoluteFill is bounded */}
                <View style={styles.mapContainer}>
                  <DeliveryMap
                    driverLat={effectiveDriverLat}
                    driverLng={effectiveDriverLng}
                    dropoffLat={tracking?.delivery?.dropoffLatitude}
                    dropoffLng={tracking?.delivery?.dropoffLongitude}
                    chefLat={tracking?.chef?.latitude}
                    chefLng={tracking?.chef?.longitude}
                  />
                  {/* Expand chevron badge — top-right corner so it doesn't obscure the map */}
                  <View style={styles.expandBadge}>
                    <ChevronRight
                      size={14}
                      color={customerColors.charcoal.DEFAULT}
                      accessibilityElementsHidden
                    />
                  </View>
                </View>

                {/* Status / ETA row — below the map, inside the card */}
                <View style={styles.mapStatusRow}>
                  <View style={styles.mapStatusLeft}>
                    {order.status === 'preparing' ? (
                      <CookingIndicator
                        size={14}
                        color={customerColors.coral.DEFAULT}
                      />
                    ) : null}
                    <Text style={styles.mapStatusText}>
                      {getInlineStatusLabel(order.status)}
                    </Text>
                  </View>
                  {order.estimatedDeliveryTime ? (
                    <Text style={styles.mapEtaText}>
                      ~{order.estimatedDeliveryTime}
                    </Text>
                  ) : null}
                </View>
              </View>
            </Pressable>
          </View>
        )}

        {/* Leave a review — primary action once the order is delivered (#145). */}
        {order.status === 'delivered' && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={() => router.push(`/order/${order.id}/review`)}
              accessibilityRole="button"
              accessibilityLabel="Leave a review for this order"
            >
              <View style={styles.trackButton}>
                <Text style={styles.trackButtonText}>Leave a Review</Text>
              </View>
            </Pressable>
            {/* Tip your chef / rider (#45) — 100% pass-through. */}
            <Pressable
              onPress={() => router.push(`/order/${order.id}/tip`)}
              accessibilityRole="button"
              accessibilityLabel="Tip your chef or rider"
              style={{ marginTop: 12 }}
            >
              <View style={styles.tipButton}>
                <Text style={styles.tipButtonText}>Tip your chef / rider</Text>
              </View>
            </Pressable>
            {/* Reorder (#238) — re-add these items to the cart. */}
            <Pressable
              onPress={handleReorder}
              disabled={reorder.isPending}
              accessibilityRole="button"
              accessibilityLabel="Reorder these items"
              style={{ marginTop: 12 }}
            >
              <View style={styles.reorderButton}>
                {reorder.isPending ? (
                  <ActivityIndicator color={customerColors.coral.DEFAULT} />
                ) : (
                  <Text style={styles.reorderButtonText}>Reorder</Text>
                )}
              </View>
            </Pressable>
          </View>
        )}

        {/* Report an issue (#37) — on a paid, non-cancelled order (active or
            delivered). Mirrors the API guard. The server decides the
            instant/assisted refund. */}
        {order.paymentStatus === 'completed' && order.status !== 'cancelled' && (
          <View style={styles.reportWrapper}>
            {/* Message support about this order (#53) — admin-mediated chat. */}
            <Pressable
              onPress={() => router.push(`/order/${order.id}/messages` as never)}
              accessibilityRole="button"
              accessibilityLabel="Message support about this order"
            >
              <Text style={styles.reportLink}>Message support about this order</Text>
            </Pressable>
            <Pressable
              onPress={() => router.push(`/order/${order.id}/report-issue` as never)}
              accessibilityRole="button"
              accessibilityLabel="Report an issue with this order"
              style={{ marginTop: 10 }}
            >
              <Text style={styles.reportLink}>Report an issue with this order</Text>
            </Pressable>
          </View>
        )}

        {/* Reorder — primary action for a cancelled order (#238). */}
        {order.status === 'cancelled' && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={handleReorder}
              disabled={reorder.isPending}
              accessibilityRole="button"
              accessibilityLabel="Reorder these items"
            >
              <View style={styles.trackButton}>
                {reorder.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.trackButtonText}>Reorder</Text>
                )}
              </View>
            </Pressable>
          </View>
        )}

        {/* Items list — clean rows separated by hairline, tabular prices */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Items</Text>
          {order.items.map((item, index) => (
            <View
              key={String(index)}
              style={[
                styles.itemRow,
                index < order.items.length - 1 && styles.itemRowDivider,
              ]}
            >
              <View style={styles.itemInfo}>
                <Text style={styles.itemName} numberOfLines={2}>
                  {item.name}
                </Text>
                <Text style={styles.itemQty}>×{item.quantity}</Text>
              </View>
              <Text style={styles.itemSubtotal}>
                ₹{(item.price * item.quantity).toFixed(2)}
              </Text>
            </View>
          ))}
        </View>

        {/* Hairline divider between sections */}
        <View style={styles.sectionDivider} />

        {/* Delivery address */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Delivery Address</Text>
          <Text style={styles.addressText}>
            {order.deliveryAddress.addressLine1}
          </Text>
          {order.deliveryAddress.addressLine2 ? (
            <Text style={styles.addressText}>
              {order.deliveryAddress.addressLine2}
            </Text>
          ) : null}
          <Text style={styles.addressText}>
            {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
            {order.deliveryAddress.pincode}
          </Text>
        </View>

        {/* Hairline divider */}
        <View style={styles.sectionDivider} />

        {/* Price breakdown — tabular-nums for all monetary figures */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Price Breakdown</Text>
          <View style={styles.priceRow}>
            <Text style={styles.priceLabel}>Subtotal</Text>
            <Text style={styles.priceValue}>₹{subtotal.toFixed(2)}</Text>
          </View>
          <View style={styles.priceRow}>
            <Text style={styles.priceLabel}>Delivery Fee</Text>
            <Text style={styles.priceValue}>₹{deliveryFee.toFixed(2)}</Text>
          </View>
          {/* Total row — hairline rule above, heavier weight */}
          <View style={[styles.priceRow, styles.totalRow]}>
            <Text style={styles.totalLabel}>Total</Text>
            <Text style={styles.totalValue}>
              ₹{order.totalAmount.toFixed(2)}
            </Text>
          </View>
        </View>

        {/* Order date — small caption, centred */}
        <View style={styles.dateSection}>
          <Text style={styles.dateText}>
            Ordered on {formatDateTime(order.createdAt)}
          </Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  // White-canvas root — spec §1
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

  // Ghost / outline button for error state
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
    paddingBottom: 48,
  },

  // Header section — chef name + order number; hairline divider below
  headerSection: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },

  // Circular back button — white bg, hairline border, shadow[2]
  backCircle: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
    // shadow[2]
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  headerText: {
    flex: 1,
  },
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.2,
    marginBottom: 2,
  },
  orderNumber: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },

  // Status chip + ETA row
  statusSection: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  statusChipRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  // Spec §2.7: radius-full chip, tint bg + family text color
  statusChip: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 9999,
    alignSelf: 'flex-start',
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
  paymentPendingText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
    marginTop: 4,
  },

  // ---- Inline map card (replaces Track Order button) -------------------------
  // Outer wrapper — horizontal padding + bottom gap before the items section
  mapCardWrapper: {
    paddingHorizontal: 16,
    paddingBottom: 16,
  },
  // Card shell — radius 16, hairline border, white bg, overflow hidden so the
  // map clips to rounded corners. Shadow[2] lifts it off the canvas.
  mapCard: {
    borderRadius: 16,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    overflow: 'hidden',
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
  },
  // Fixed-height map container — DeliveryMap uses StyleSheet.absoluteFill
  // so it must be bounded by an explicit height parent.
  mapContainer: {
    height: 210,
    position: 'relative',
  },
  // Expand chevron — white pill badge, top-right corner of the map
  expandBadge: {
    position: 'absolute',
    top: 10,
    right: 10,
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: customerColors.surface.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.10,
    shadowRadius: 4,
    elevation: 2,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
  },
  // Status / ETA footer row — inside the card, below the map
  mapStatusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  mapStatusLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    flex: 1,
  },
  // Status label — Inter body, charcoal-soft
  mapStatusText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    flexShrink: 1,
  },
  // ETA — tabular-nums, coral-pressed for emphasis
  mapEtaText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.pressed,
    fontVariant: ['tabular-nums'],
    marginLeft: 8,
  },

  // CTA container — horizontal padding
  ctaWrapper: {
    paddingHorizontal: 16,
    paddingBottom: 8,
  },
  // Spec §3 primary button: coral fill, radius 8, minHeight 52, SemiBold canvas text
  trackButton: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  trackButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.canvas,
  },
  // Secondary CTA (coral outline) — tip sits below the filled review button.
  tipButton: {
    backgroundColor: customerColors.canvas,
    borderRadius: 8,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
  },
  tipButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.coral.DEFAULT,
  },
  // Reorder (#238) — coral-outline secondary, same footprint as the tip button.
  reorderButton: {
    backgroundColor: customerColors.canvas,
    borderRadius: 8,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
  },
  reorderButtonText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.coral.DEFAULT,
  },
  // Report an issue (#37) — quiet text link below the primary actions.
  reportWrapper: { paddingHorizontal: 20, marginTop: 16, alignItems: 'center' },
  reportLink: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textDecorationLine: 'underline',
  },

  // Content section — direct on white, no card bg
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

  // Hairline rule between major sections (spec §1: separation by hairline)
  sectionDivider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginHorizontal: 16,
  },

  // Item rows — clean, separated by inset hairlines
  itemRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 10,
  },
  itemRowDivider: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  itemInfo: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginRight: 12,
  },
  itemName: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    flex: 1,
  },
  itemQty: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  itemSubtotal: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  // Delivery address text
  addressText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 22,
  },

  // Price breakdown rows — tabular-nums on all monetary values
  priceRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
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
  // Total row — hairline above, heavier weight
  totalRow: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    marginTop: 8,
    paddingTop: 12,
    marginBottom: 0,
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

  // Order date caption — centred, charcoal-soft
  dateSection: {
    paddingHorizontal: 16,
    paddingTop: 16,
    alignItems: 'center',
  },
  dateText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
});
