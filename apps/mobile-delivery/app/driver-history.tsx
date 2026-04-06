import { useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

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

const PAGE_LIMIT = 20;

function StatusBadge({ status }: { status: string }) {
  const isDelivered = status === 'delivered';
  const isCancelled = status === 'cancelled';
  const bgColor = isDelivered
    ? 'bg-green-100'
    : isCancelled
    ? 'bg-red-100'
    : 'bg-yellow-100';
  const textColor = isDelivered
    ? 'text-green-700'
    : isCancelled
    ? 'text-red-700'
    : 'text-yellow-700';
  const label = status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ');
  return (
    <View className={`px-2 py-0.5 rounded-full ${bgColor}`}>
      <Text className={`text-xs font-medium ${textColor}`}>{label}</Text>
    </View>
  );
}

function HistoryCard({
  item,
  onPress,
}: {
  item: DeliveryHistoryItem;
  onPress: () => void;
}) {
  const date = new Date(item.completedAt).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });

  return (
    <TouchableOpacity
      onPress={onPress}
      className="bg-white rounded-2xl p-4 mb-3 shadow-sm"
      activeOpacity={0.75}
    >
      <View className="flex-row items-center justify-between mb-2">
        <StatusBadge status={item.status} />
        <Text className="text-xs text-gray-400">{date}</Text>
      </View>

      <View className="flex-row items-center mb-1">
        <View className="w-2 h-2 rounded-full bg-orange-400 mr-2" />
        <Text className="flex-1 text-sm text-gray-700" numberOfLines={1}>
          {item.pickupAddress}
        </Text>
      </View>
      <View className="flex-row items-center mb-3">
        <View className="w-2 h-2 rounded-full bg-green-500 mr-2" />
        <Text className="flex-1 text-sm text-gray-700" numberOfLines={1}>
          {item.dropoffAddress}
        </Text>
      </View>

      <View className="flex-row justify-between border-t border-gray-100 pt-2">
        <Text className="text-sm text-gray-500">{item.distance.toFixed(1)} km</Text>
        <Text className="text-sm font-semibold text-orange-500">
          &#8377;{item.payout.toFixed(2)}
        </Text>
      </View>
    </TouchableOpacity>
  );
}

function SkeletonCard() {
  return (
    <View className="bg-gray-200 rounded-2xl p-4 mb-3 h-28 animate-pulse" />
  );
}

export default function DriverHistoryScreen() {
  const [isFetchingMore, setIsFetchingMore] = useState(false);

  const {
    data,
    isLoading,
    isError,
    refetch,
    isRefetching,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery<DeliveryHistoryResponse>({
    queryKey: ['driver', 'history'],
    queryFn: async ({ pageParam }) => {
      const page = typeof pageParam === 'number' ? pageParam : 1;
      const r = await api.get('/delivery/orders', {
        params: { page, limit: PAGE_LIMIT },
      });
      return r.data as DeliveryHistoryResponse;
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const fetched = (lastPage.page - 1) * PAGE_LIMIT + lastPage.deliveries.length;
      return fetched < lastPage.total ? lastPage.page + 1 : undefined;
    },
  });

  const deliveries =
    data?.pages.flatMap((page) => page.deliveries) ?? [];

  function handleEndReached() {
    if (hasNextPage && !isFetchingNextPage && !isFetchingMore) {
      setIsFetchingMore(true);
      fetchNextPage().finally(() => setIsFetchingMore(false));
    }
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50">
        <View className="px-4 pt-4 pb-2">
          <Text className="text-2xl font-bold text-gray-900">Delivery History</Text>
        </View>
        <View className="px-4 mt-2">
          {[1, 2, 3].map((i) => (
            <SkeletonCard key={i} />
          ))}
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-gray-500 text-base mb-4">
          Failed to load delivery history
        </Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-orange-500 px-6 py-3 rounded-xl"
        >
          <Text className="text-white font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <View className="px-4 pt-4 pb-2">
        <Text className="text-2xl font-bold text-gray-900">Delivery History</Text>
      </View>

      <FlatList<DeliveryHistoryItem>
        data={deliveries}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 24 }}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor="#FF6B35"
          />
        }
        onEndReached={handleEndReached}
        onEndReachedThreshold={0.3}
        ListEmptyComponent={
          <View className="items-center justify-center py-16">
            <Text className="text-4xl mb-4">&#128230;</Text>
            <Text className="text-base text-gray-500 text-center">
              No deliveries yet.{'\n'}Your completed deliveries will appear here.
            </Text>
          </View>
        }
        ListFooterComponent={
          isFetchingNextPage ? (
            <View className="py-4 items-center">
              <ActivityIndicator size="small" color="#FF6B35" />
            </View>
          ) : null
        }
        renderItem={({ item }) => (
          <HistoryCard
            item={item}
            onPress={() => router.push(`/delivery/${item.id}`)}
          />
        )}
      />
    </SafeAreaView>
  );
}
