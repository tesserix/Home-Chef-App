import { useState } from 'react';
import { View, Text, FlatList, Pressable, RefreshControl } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useVendorPendingOrders, useVendorOrderHistory, useOrderAction } from '../../hooks/useVendorOrders';
import { OrderCard } from '../../components/vendor/OrderCard';
import { UndoSnackbar } from '../../components/vendor/UndoSnackbar';
import type { Order } from '../../hooks/useVendorOrders';

type ActiveTab = 'live' | 'history';

function SkeletonCard() {
  return (
    <View className="mb-3 rounded-2xl bg-gray-200 h-36 animate-pulse" />
  );
}

function EmptyQueue() {
  return (
    <View className="flex-1 items-center justify-center px-8 py-16">
      <View className="mb-4 h-16 w-16 rounded-full bg-orange-50 items-center justify-center">
        <Text className="text-3xl">🍳</Text>
      </View>
      <Text className="text-center text-base font-semibold text-gray-700">
        No pending orders
      </Text>
      <Text className="mt-2 text-center text-sm text-gray-400">
        New orders will appear here automatically
      </Text>
    </View>
  );
}

function HistoryOrderRow({ order }: { order: Order }) {
  const statusColor: Record<string, string> = {
    delivered: 'text-green-600',
    cancelled: 'text-red-500',
    rejected: 'text-red-500',
    picked_up: 'text-teal-600',
  };
  const colorClass = statusColor[order.status] ?? 'text-gray-500';
  const itemCount = order.items.length;

  return (
    <View className="mb-2 rounded-xl bg-white px-4 py-3 border border-gray-100">
      <View className="flex-row items-center justify-between">
        <Text className="text-sm font-semibold text-gray-900">{order.customerName}</Text>
        <Text className={`text-xs font-medium capitalize ${colorClass}`}>
          {order.status.replace('_', ' ')}
        </Text>
      </View>
      <View className="mt-1 flex-row items-center justify-between">
        <Text className="text-xs text-gray-400">
          {itemCount} item{itemCount !== 1 ? 's' : ''}
        </Text>
        <Text className="text-sm font-bold text-gray-800">₹{order.total.toFixed(0)}</Text>
      </View>
    </View>
  );
}

function LiveQueue() {
  const { data, isLoading } = useVendorPendingOrders();
  const { triggerAction, handleUndo, pendingUndo, isLoading: actionLoading } = useOrderAction();
  const orders = data?.orders ?? [];

  if (isLoading) {
    return (
      <View className="flex-1 px-4 pt-4">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </View>
    );
  }

  return (
    <View className="flex-1">
      <FlatList
        data={orders}
        keyExtractor={(item) => item.id}
        contentContainerClassName="px-4 pt-4 pb-24"
        renderItem={({ item }) => (
          <OrderCard
            order={item}
            onAccept={() => triggerAction(item.id, 'accepted')}
            onReject={() => triggerAction(item.id, 'rejected')}
            isLoading={actionLoading}
          />
        )}
        ListEmptyComponent={<EmptyQueue />}
      />
      <UndoSnackbar pendingUndo={pendingUndo} onUndo={handleUndo} />
    </View>
  );
}

function HistoryList() {
  const [page, setPage] = useState(1);
  const { data, isLoading, isRefetching, refetch } = useVendorOrderHistory(page);
  const orders = data?.orders ?? [];
  const hasMore = data ? page * (data.limit ?? 20) < data.total : false;

  if (isLoading) {
    return (
      <View className="flex-1 px-4 pt-4">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </View>
    );
  }

  return (
    <FlatList
      data={orders}
      keyExtractor={(item) => item.id}
      contentContainerClassName="px-4 pt-4 pb-8"
      renderItem={({ item }) => <HistoryOrderRow order={item} />}
      refreshControl={
        <RefreshControl
          refreshing={isRefetching}
          onRefresh={() => { setPage(1); refetch(); }}
          tintColor="#f97316"
        />
      }
      onEndReachedThreshold={0.3}
      onEndReached={() => {
        if (hasMore) setPage((p) => p + 1);
      }}
      ListEmptyComponent={
        <View className="flex-1 items-center justify-center py-16">
          <Text className="text-sm text-gray-400">No order history yet</Text>
        </View>
      }
    />
  );
}

export default function OrdersScreen() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('live');

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* Header */}
      <View className="px-4 pt-4 pb-2">
        <Text className="text-2xl font-bold text-gray-900">Orders</Text>
      </View>

      {/* Segmented control */}
      <View className="mx-4 mb-3 flex-row rounded-xl bg-gray-100 p-1">
        <Pressable
          onPress={() => setActiveTab('live')}
          className={`flex-1 rounded-lg py-2.5 items-center ${
            activeTab === 'live' ? 'bg-orange-500' : 'bg-transparent'
          }`}
        >
          <Text
            className={`text-sm font-semibold ${
              activeTab === 'live' ? 'text-white' : 'text-gray-600'
            }`}
          >
            Live Queue
          </Text>
        </Pressable>
        <Pressable
          onPress={() => setActiveTab('history')}
          className={`flex-1 rounded-lg py-2.5 items-center ${
            activeTab === 'history' ? 'bg-orange-500' : 'bg-transparent'
          }`}
        >
          <Text
            className={`text-sm font-semibold ${
              activeTab === 'history' ? 'text-white' : 'text-gray-600'
            }`}
          >
            History
          </Text>
        </Pressable>
      </View>

      {/* Content */}
      {activeTab === 'live' ? <LiveQueue /> : <HistoryList />}
    </SafeAreaView>
  );
}
