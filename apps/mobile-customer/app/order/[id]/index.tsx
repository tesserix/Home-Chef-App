import React from 'react';
import {
  ActivityIndicator,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useOrder } from '../../../hooks/useOrderHistory';
import type { Order } from '../../../types/customer';

const ACTIVE_STATUSES: Order['status'][] = [
  'confirmed',
  'preparing',
  'ready',
  'picked_up',
];

type StatusColor = { bg: string; text: string; label: string };

function getStatusStyle(status: Order['status']): StatusColor {
  switch (status) {
    case 'pending':
      return { bg: '#FEF9C3', text: '#92400E', label: 'Pending' };
    case 'confirmed':
      return { bg: '#DBEAFE', text: '#1E40AF', label: 'Confirmed' };
    case 'preparing':
      return { bg: '#EDE9FE', text: '#5B21B6', label: 'Preparing' };
    case 'ready':
      return { bg: '#DCFCE7', text: '#166534', label: 'Ready for Pickup' };
    case 'picked_up':
      return { bg: '#DCFCE7', text: '#166534', label: 'On the Way' };
    case 'delivered':
      return { bg: '#F0FDF4', text: '#3e6b3c', label: 'Delivered' };
    case 'cancelled':
      return { bg: '#f3dcd2', text: '#991B1B', label: 'Cancelled' };
    default:
      return { bg: '#e6e5e0', text: '#4a4a47', label: status };
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
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#3e6b3c" />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !data?.data) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <Text style={styles.errorText}>Order not found</Text>
          <TouchableOpacity
            onPress={() => router.back()}
            style={styles.backButton}
          >
            <Text style={styles.backButtonText}>Go Back</Text>
          </TouchableOpacity>
        </View>
      </SafeAreaView>
    );
  }

  const order = data.data;
  const statusStyle = getStatusStyle(order.status);
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
    <SafeAreaView style={styles.container}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Header */}
        <View style={styles.headerSection}>
          <Text style={styles.chefName}>{order.chef.name}</Text>
          <Text style={styles.orderNumber}>Order #{order.orderNumber}</Text>
        </View>

        {/* Status badge */}
        <View style={styles.statusSection}>
          <View
            style={[
              styles.statusBadge,
              { backgroundColor: statusStyle.bg },
            ]}
          >
            <Text style={[styles.statusText, { color: statusStyle.text }]}>
              {statusStyle.label}
            </Text>
          </View>
          {order.estimatedDeliveryTime && (
            <Text style={styles.etaText}>
              ETA: {order.estimatedDeliveryTime}
            </Text>
          )}
        </View>

        {/* Track Order button — only for active orders */}
        {isActiveOrder && (
          <TouchableOpacity
            style={styles.trackButton}
            onPress={handleTrackOrder}
            activeOpacity={0.8}
          >
            <Text style={styles.trackButtonText}>Track Order</Text>
          </TouchableOpacity>
        )}

        {/* Order items */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Items</Text>
          {order.items.map((item, index) => (
            <View key={String(index)} style={styles.itemRow}>
              <View style={styles.itemInfo}>
                <Text style={styles.itemName}>{item.name}</Text>
                <Text style={styles.itemQty}>x{item.quantity}</Text>
              </View>
              <Text style={styles.itemSubtotal}>
                ₹{(item.price * item.quantity).toFixed(2)}
              </Text>
            </View>
          ))}
        </View>

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

        {/* Price breakdown */}
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
          <View style={[styles.priceRow, styles.totalRow]}>
            <Text style={styles.totalLabel}>Total</Text>
            <Text style={styles.totalValue}>
              ₹{order.totalAmount.toFixed(2)}
            </Text>
          </View>
        </View>

        {/* Order date */}
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
  container: {
    flex: 1,
    backgroundColor: '#fafaf7',
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  errorText: {
    fontSize: 16,
    color: '#7a7a76',
    marginBottom: 16,
  },
  backButton: {
    paddingHorizontal: 24,
    paddingVertical: 10,
    backgroundColor: '#3e6b3c',
    borderRadius: 8,
  },
  backButtonText: {
    color: '#fafaf7',
    fontWeight: '600',
  },
  scrollContent: {
    paddingBottom: 32,
  },
  headerSection: {
    backgroundColor: '#fafaf7',
    paddingHorizontal: 16,
    paddingTop: 20,
    paddingBottom: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#e6e5e0',
  },
  chefName: {
    fontSize: 22,
    fontWeight: '700',
    color: '#1a1a18',
    marginBottom: 4,
  },
  orderNumber: {
    fontSize: 14,
    color: '#7a7a76',
  },
  statusSection: {
    backgroundColor: '#fafaf7',
    paddingHorizontal: 16,
    paddingVertical: 12,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 8,
  },
  statusBadge: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 20,
  },
  statusText: {
    fontSize: 14,
    fontWeight: '600',
  },
  etaText: {
    fontSize: 13,
    color: '#7a7a76',
  },
  trackButton: {
    marginHorizontal: 16,
    marginBottom: 8,
    backgroundColor: '#3e6b3c',
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
  },
  trackButtonText: {
    color: '#fafaf7',
    fontSize: 16,
    fontWeight: '700',
  },
  section: {
    backgroundColor: '#fafaf7',
    marginTop: 8,
    paddingHorizontal: 16,
    paddingVertical: 16,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
    marginBottom: 12,
  },
  itemRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 10,
  },
  itemInfo: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  itemName: {
    fontSize: 14,
    color: '#4a4a47',
    flex: 1,
  },
  itemQty: {
    fontSize: 13,
    color: '#7a7a76',
  },
  itemSubtotal: {
    fontSize: 14,
    fontWeight: '600',
    color: '#1a1a18',
  },
  addressText: {
    fontSize: 14,
    color: '#4a4a47',
    lineHeight: 22,
  },
  priceRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  priceLabel: {
    fontSize: 14,
    color: '#7a7a76',
  },
  priceValue: {
    fontSize: 14,
    color: '#4a4a47',
  },
  totalRow: {
    borderTopWidth: 1,
    borderTopColor: '#e6e5e0',
    marginTop: 8,
    paddingTop: 12,
  },
  totalLabel: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
  },
  totalValue: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
  },
  dateSection: {
    paddingHorizontal: 16,
    paddingTop: 16,
    alignItems: 'center',
  },
  dateText: {
    fontSize: 13,
    color: '#7a7a76',
  },
});
