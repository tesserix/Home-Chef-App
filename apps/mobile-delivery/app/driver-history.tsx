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
    ? 'bg-herb-tint'
    : isCancelled
    ? 'bg-paprika-tint'
    : 'bg-amber-tint';
  const textColor = isDelivered
    ? 'text-herb'
    : isCancelled
    ? 'text-paprika'
    : 'text-amber';
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
      className="bg-bone rounded-2xl p-4 mb-3 shadow-sm"
      activeOpacity={0.75}
    >
      <View className="flex-row items-center justify-between mb-2">
        <StatusBadge status={item.status} />
        <Text className="text-xs text-ink-muted">{date}</Text>
      </View>

      <View className="flex-row items-center mb-1">
        <View className="w-2 h-2 rounded-full bg-herb-soft mr-2" />
        <Text className="flex-1 text-sm text-ink-soft" numberOfLines={1}>
          {item.pickupAddress}
        </Text>
      </View>
      <View className="flex-row items-center mb-3">
        <View className="w-2 h-2 rounded-full bg-herb mr-2" />
        <Text className="flex-1 text-sm text-ink-soft" numberOfLines={1}>
          {item.dropoffAddress}
        </Text>
      </View>

      <View className="flex-row justify-between border-t border-mist pt-2">
        <Text className="text-sm text-ink-muted">{item.distance.toFixed(1)} km</Text>
        <Text className="text-sm font-semibold text-herb">
          &#8377;{item.payout.toFixed(2)}
        </Text>
      </View>
    </TouchableOpacity>
  );
}

function SkeletonCard() {
  return (
    <View className="bg-mist rounded-2xl p-4 mb-3 h-28 animate-pulse" />
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
      <SafeAreaView className="flex-1 bg-paper">
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">Delivery History</Text>
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
      <SafeAreaView className="flex-1 bg-paper items-center justify-center px-6">
        <Text className="text-ink-muted text-base mb-4">
          Failed to load delivery history
        </Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-herb px-6 py-3 rounded-xl"
        >
          <Text className="text-paper font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <View className="px-4 pt-4 pb-2">
        <Text className="font-display text-2xl font-semibold text-ink">Delivery History</Text>
      </View>

      <FlatList<DeliveryHistoryItem>
        data={deliveries}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 24 }}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor="#3e6b3c"
          />
        }
        onEndReached={handleEndReached}
        onEndReachedThreshold={0.3}
        ListEmptyComponent={
          <View className="items-center justify-center py-16">
            <Text className="text-4xl mb-4">&#128230;</Text>
            <Text className="text-base text-ink-muted text-center">
              No deliveries yet.{'\n'}Your completed deliveries will appear here.
            </Text>
          </View>
        }
        ListFooterComponent={
          isFetchingNextPage ? (
            <View className="py-4 items-center">
              <ActivityIndicator size="small" color="#3e6b3c" />
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
