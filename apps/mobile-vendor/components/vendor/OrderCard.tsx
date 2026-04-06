import { View, Text, Pressable } from 'react-native';
import { formatDistanceToNow } from 'date-fns';
import type { Order } from '../../hooks/useVendorOrders';

interface OrderCardProps {
  order: Order;
  onAccept: () => void;
  onReject: () => void;
  isLoading?: boolean;
}

export function OrderCard({ order, onAccept, onReject, isLoading = false }: OrderCardProps) {
  const elapsedTime = formatDistanceToNow(new Date(order.createdAt), { addSuffix: true });
  const itemsSummary = order.items
    .map((item) => `${item.name} x${item.quantity}`)
    .join(', ');

  return (
    <View className="mb-3 w-full rounded-2xl bg-white p-4 shadow-md">
      {/* Row 1: Customer name + elapsed time */}
      <View className="flex-row items-center justify-between">
        <Text className="text-lg font-semibold text-gray-900">{order.customerName}</Text>
        <Text className="text-sm text-gray-400">{elapsedTime}</Text>
      </View>

      {/* Row 2: Items list */}
      <Text className="mt-2 text-sm text-gray-600" numberOfLines={2}>
        {itemsSummary}
      </Text>

      {/* Row 3: Total */}
      <Text className="mt-2 text-base font-bold text-gray-900">
        ₹{order.total.toFixed(0)}
      </Text>

      {/* Row 4: Special instructions */}
      {order.specialInstructions ? (
        <Text className="mt-1 text-xs italic text-gray-500" numberOfLines={2}>
          {order.specialInstructions}
        </Text>
      ) : null}

      {/* Row 5: Accept / Reject buttons */}
      <View className="mt-4 flex-row gap-3">
        <Pressable
          onPress={onReject}
          disabled={isLoading}
          className="flex-1 rounded-xl border border-red-500 py-3 items-center active:opacity-70 disabled:opacity-40"
        >
          <Text className="text-sm font-semibold text-red-500">Reject</Text>
        </Pressable>
        <Pressable
          onPress={onAccept}
          disabled={isLoading}
          className="flex-1 rounded-xl bg-green-500 py-3 items-center active:opacity-70 disabled:opacity-40"
        >
          <Text className="text-sm font-semibold text-white">Accept</Text>
        </Pressable>
      </View>
    </View>
  );
}
