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
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';
import type { Order } from '../../../types/customer';

const ACTIVE_STATUSES: Order['status'][] = [
  'confirmed',
  'preparing',
  'ready',
  'picked_up',
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
        label: 'Ready for Pickup',
      };
    case 'picked_up':
      return {
        bg: customerColors.coral.tint,
        text: customerColors.coral.pressed,
        label: 'On the Way',
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

export default function OrderDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, isError } = useOrder(id ?? '');

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

  const subtotal = order.items.reduce(
    (sum, item) => sum + item.price * item.quantity,
    0,
  );
  const deliveryFee = order.totalAmount - subtotal;

  function handleTrackOrder() {
    router.push(`/order/${order.id}/track`);
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
            <Text style={styles.chefName}>{order.chef.name}</Text>
            <Text style={styles.orderNumber}>Order #{order.orderNumber}</Text>
          </View>
        </View>

        {/* Status chip + ETA — spec §2.7 + §0 chip pattern */}
        <View style={styles.statusSection}>
          <View
            style={[styles.statusChip, { backgroundColor: chipStyle.bg }]}
          >
            <Text style={[styles.statusChipText, { color: chipStyle.text }]}>
              {chipStyle.label}
            </Text>
          </View>
          {order.estimatedDeliveryTime ? (
            <Text style={styles.etaText}>
              ETA: {order.estimatedDeliveryTime}
            </Text>
          ) : null}
        </View>

        {/* Track Order CTA — coral filled, only for active/in-flight orders.
            Spec §2.6: coral when the order is in-flight.
            iOS Pressable bug: visual styles on inner View. */}
        {isActiveOrder && (
          <View style={styles.ctaWrapper}>
            <Pressable
              onPress={handleTrackOrder}
              accessibilityRole="button"
              accessibilityLabel="Track this order"
            >
              <View style={styles.trackButton}>
                <Text style={styles.trackButtonText}>Track Order</Text>
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

  // Track Order CTA container — horizontal padding
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
