import { ActivityIndicator, ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { api } from '../../lib/api';

interface DeliveryHistoryItem {
  id: string;
  orderId: string;
  status: string;
  pickupAddress: string;
  dropoffAddress: string;
  payout: number;
  completedAt: string;
  distance: number;
}

interface DeliveryHistoryResponse {
  deliveries: DeliveryHistoryItem[];
  total: number;
  page: number;
  limit: number;
}

function StatusBadge({ status }: { status: string }) {
  const isDelivered = status === 'delivered';
  const isCancelled = status === 'cancelled';
  const bgColor = isDelivered ? 'bg-herb-tint' : isCancelled ? 'bg-paprika-tint' : 'bg-amber-tint';
  const textColor = isDelivered ? 'text-herb' : isCancelled ? 'text-paprika' : 'text-amber';
  const label = status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ');
  return (
    <View className={`px-3 py-1 rounded-full ${bgColor}`}>
      <Text className={`text-sm font-medium ${textColor}`}>{label}</Text>
    </View>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-row justify-between items-start py-3 border-b border-mist">
      <Text className="text-sm text-ink-muted w-1/3">{label}</Text>
      <Text className="text-sm text-ink flex-1 text-right">{value}</Text>
    </View>
  );
}

export default function DeliveryDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();

  const { data: delivery, isLoading, isError, refetch } = useQuery<DeliveryHistoryItem | null>({
    queryKey: ['driver', 'history', 'delivery', id],
    queryFn: async () => {
      const r = await api.get<DeliveryHistoryResponse>('/delivery/orders', {
        params: { deliveryId: id, limit: 1 },
      });
      return r.data.deliveries[0] ?? null;
    },
    enabled: !!id,
  });

  function handleBack() {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/driver-history');
    }
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-4 pb-2">
        <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={handleBack} className="mr-3 p-3" activeOpacity={0.7}>
          <ChevronLeft size={24} color="#4a4a47" />
        </TouchableOpacity>
        <Text className="text-xl font-semibold text-ink">Delivery Detail</Text>
      </View>

      {isLoading && (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color="#3e6b3c" />
        </View>
      )}

      {isError && (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-ink-muted text-base mb-4">Failed to load delivery detail</Text>
          <TouchableOpacity onPress={() => refetch()} className="bg-herb px-6 py-3 rounded-xl">
            <Text className="text-paper font-semibold">Retry</Text>
          </TouchableOpacity>
        </View>
      )}

      {!isLoading && !isError && !delivery && (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-ink-muted text-base">Delivery not found.</Text>
        </View>
      )}

      {delivery && (
        <ScrollView className="flex-1 px-4" contentContainerStyle={{ paddingBottom: 32 }}>
          {/* Status & Payout card */}
          <View className="bg-bone rounded-2xl p-4 mt-4 shadow-sm">
            <View className="flex-row items-center justify-between mb-4">
              <StatusBadge status={delivery.status} />
              <Text className="font-display text-2xl font-semibold text-herb">
                &#8377;{delivery.payout.toFixed(2)}
              </Text>
            </View>
            <Text className="text-xs text-ink-muted">
              Order #{delivery.orderId.slice(-8).toUpperCase()}
            </Text>
          </View>

          {/* Addresses */}
          <View className="bg-bone rounded-2xl p-4 mt-3 shadow-sm">
            <Text className="text-sm font-semibold text-ink-soft mb-3">Route</Text>
            <View className="flex-row items-start mb-3">
              <View className="w-3 h-3 rounded-full bg-herb-soft mt-1 mr-3" />
              <View className="flex-1">
                <Text className="text-xs text-ink-muted mb-0.5">Pickup</Text>
                <Text className="text-sm text-ink">{delivery.pickupAddress}</Text>
              </View>
            </View>
            <View className="flex-row items-start">
              <View className="w-3 h-3 rounded-full bg-herb mt-1 mr-3" />
              <View className="flex-1">
                <Text className="text-xs text-ink-muted mb-0.5">Drop-off</Text>
                <Text className="text-sm text-ink">{delivery.dropoffAddress}</Text>
              </View>
            </View>
          </View>

          {/* Details */}
          <View className="bg-bone rounded-2xl px-4 mt-3 shadow-sm">
            <DetailRow label="Distance" value={`${delivery.distance.toFixed(1)} km`} />
            <DetailRow
              label="Completed"
              value={new Date(delivery.completedAt).toLocaleString('en-IN', {
                day: 'numeric',
                month: 'short',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            />
            <DetailRow label="Payout" value={`\u20B9${delivery.payout.toFixed(2)}`} />
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
