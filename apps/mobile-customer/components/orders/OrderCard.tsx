import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useRouter } from 'expo-router';
import type { Order } from '../../types/customer';

interface OrderCardProps {
  order: Order;
}

type StatusColor = {
  bg: string;
  text: string;
  label: string;
};

function getStatusStyle(status: Order['status']): StatusColor {
  switch (status) {
    case 'pending':
      return { bg: '#FEF9C3', text: '#92400E', label: 'Pending' };
    case 'confirmed':
      return { bg: '#DBEAFE', text: '#1E40AF', label: 'Confirmed' };
    case 'preparing':
      return { bg: '#EDE9FE', text: '#5B21B6', label: 'Preparing' };
    case 'ready':
      return { bg: '#DCFCE7', text: '#166534', label: 'Ready' };
    case 'picked_up':
      return { bg: '#DCFCE7', text: '#166534', label: 'Picked Up' };
    case 'delivered':
      return { bg: '#F0FDF4', text: '#C2410C', label: 'Delivered' };
    case 'cancelled':
      return { bg: '#f3dcd2', text: '#991B1B', label: 'Cancelled' };
    default:
      return { bg: '#e6e5e0', text: '#4a4a47', label: status };
  }
}

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
  const statusStyle = getStatusStyle(order.status);
  const itemCount = order.items.reduce((sum, item) => sum + item.quantity, 0);

  function handlePress() {
    router.push(`/order/${order.id}`);
  }

  return (
    <TouchableOpacity
      style={styles.card}
      onPress={handlePress}
      activeOpacity={0.7}
    >
      <View style={styles.header}>
        <View style={styles.orderInfo}>
          <Text style={styles.orderNumber}>#{order.orderNumber}</Text>
          <Text style={styles.chefName}>{order.chef.name}</Text>
        </View>
        <View style={[styles.statusBadge, { backgroundColor: statusStyle.bg }]}>
          <Text style={[styles.statusText, { color: statusStyle.text }]}>
            {statusStyle.label}
          </Text>
        </View>
      </View>

      <View style={styles.divider} />

      <View style={styles.footer}>
        <Text style={styles.meta}>
          {itemCount} {itemCount === 1 ? 'item' : 'items'} •{' '}
          {formatDate(order.createdAt)}
        </Text>
        <Text style={styles.amount}>₹{order.totalAmount.toFixed(2)}</Text>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: '#fafaf7',
    borderRadius: 12,
    padding: 16,
    marginHorizontal: 16,
    marginVertical: 6,
    shadowColor: '#1a1a18',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 4,
    elevation: 2,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
  },
  orderInfo: {
    flex: 1,
    marginRight: 12,
  },
  orderNumber: {
    fontSize: 13,
    color: '#7a7a76',
    marginBottom: 2,
  },
  chefName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1a1a18',
  },
  statusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 20,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '600',
  },
  divider: {
    height: 1,
    backgroundColor: '#e6e5e0',
    marginVertical: 12,
  },
  footer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  meta: {
    fontSize: 13,
    color: '#7a7a76',
  },
  amount: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
  },
});
