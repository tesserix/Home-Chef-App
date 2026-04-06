import { ArrowRight, Clock, Package, User } from 'lucide-react-native';
import { ActivityIndicator, Text, TouchableOpacity, View } from 'react-native';
import { type AvailableDelivery } from '../../hooks/useDriverDeliveries';

interface DeliveryCardProps {
  delivery: AvailableDelivery;
  onAccept: () => void;
  isLoading?: boolean;
}

export function DeliveryCard({ delivery, onAccept, isLoading = false }: DeliveryCardProps) {
  return (
    <View className="bg-white rounded-2xl shadow-sm p-4 mb-3 border border-gray-100">
      {/* Row 1: Chef name + distance */}
      <View className="flex-row items-center justify-between mb-2">
        <View className="flex-row items-center gap-1">
          <User size={14} color="#6B7280" />
          <Text className="text-sm font-semibold text-gray-800">{delivery.chefName}</Text>
        </View>
        <Text className="text-sm text-gray-500">{delivery.distance.toFixed(1)} km away</Text>
      </View>

      {/* Row 2: Pickup → Dropoff addresses */}
      <View className="flex-row items-center mb-3 gap-2">
        <Text className="text-sm text-gray-600 flex-1" numberOfLines={1}>
          {delivery.pickupAddress}
        </Text>
        <ArrowRight size={14} color="#9CA3AF" />
        <Text className="text-sm text-gray-600 flex-1" numberOfLines={1}>
          {delivery.dropoffAddress}
        </Text>
      </View>

      {/* Row 3: Item count, estimated time, payout */}
      <View className="flex-row items-center justify-between mb-4">
        <View className="flex-row items-center gap-1">
          <Package size={14} color="#6B7280" />
          <Text className="text-sm text-gray-500">
            {delivery.itemCount} {delivery.itemCount === 1 ? 'item' : 'items'}
          </Text>
        </View>
        <View className="flex-row items-center gap-1">
          <Clock size={14} color="#6B7280" />
          <Text className="text-sm text-gray-500">~{delivery.estimatedTime} min</Text>
        </View>
        <Text className="text-base font-bold text-orange-500">₹{delivery.payout}</Text>
      </View>

      {/* Accept button */}
      <TouchableOpacity
        onPress={onAccept}
        disabled={isLoading}
        className={`w-full py-3 rounded-xl items-center ${isLoading ? 'bg-orange-300' : 'bg-orange-500'}`}
        activeOpacity={0.8}
      >
        {isLoading ? (
          <ActivityIndicator size="small" color="white" />
        ) : (
          <Text className="text-white font-semibold text-base">Accept Delivery</Text>
        )}
      </TouchableOpacity>
    </View>
  );
}
