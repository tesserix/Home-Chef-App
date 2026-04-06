import React, { useState, useCallback } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useOrders } from '../../hooks/useOrderHistory';
import { OrderCard } from '../../components/orders/OrderCard';
import type { Order } from '../../types/customer';

type StatusFilter = 'all' | 'active' | 'delivered' | 'cancelled';

const STATUS_FILTERS: { key: StatusFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'delivered', label: 'Delivered' },
  { key: 'cancelled', label: 'Cancelled' },
];

const STATUS_MAP: Record<StatusFilter, string | undefined> = {
  all: undefined,
  active: 'active',
  delivered: 'delivered',
  cancelled: 'cancelled',
};

const PAGE_LIMIT = 20;

export default function OrdersScreen() {
  const [activeFilter, setActiveFilter] = useState<StatusFilter>('all');
  const [page, setPage] = useState(1);
  const [allOrders, setAllOrders] = useState<Order[]>([]);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const params = {
    page,
    limit: PAGE_LIMIT,
    status: STATUS_MAP[activeFilter],
  };

  const { data, isLoading, isRefetching, refetch } = useOrders(params);

  // Merge pages into local list
  React.useEffect(() => {
    if (data?.data) {
      if (page === 1) {
        setAllOrders(data.data);
      } else {
        setAllOrders((prev) => {
          const existingIds = new Set(prev.map((o) => o.id));
          const newItems = data.data.filter((o) => !existingIds.has(o.id));
          return [...prev, ...newItems];
        });
        setIsLoadingMore(false);
      }
    }
  }, [data, page]);

  function handleFilterChange(filter: StatusFilter) {
    setActiveFilter(filter);
    setPage(1);
    setAllOrders([]);
  }

  const handleRefresh = useCallback(() => {
    setPage(1);
    setAllOrders([]);
    void refetch();
  }, [refetch]);

  function handleLoadMore() {
    if (!data || isLoadingMore) return;
    const total = data.meta?.total ?? 0;
    if (allOrders.length < total) {
      setIsLoadingMore(true);
      setPage((prev) => prev + 1);
    }
  }

  if (isLoading && page === 1) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#F97316" />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>My Orders</Text>
      </View>

      {/* Status filter chips */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.filterRow}
      >
        {STATUS_FILTERS.map((f) => (
          <TouchableOpacity
            key={f.key}
            onPress={() => handleFilterChange(f.key)}
            style={[
              styles.filterChip,
              activeFilter === f.key && styles.filterChipActive,
            ]}
          >
            <Text
              style={[
                styles.filterChipText,
                activeFilter === f.key && styles.filterChipTextActive,
              ]}
            >
              {f.label}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <FlatList
        data={allOrders}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <OrderCard order={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={handleRefresh}
            tintColor="#F97316"
          />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.3}
        ListFooterComponent={
          isLoadingMore ? (
            <ActivityIndicator
              size="small"
              color="#F97316"
              style={styles.footerLoader}
            />
          ) : null
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyIcon}>🍽️</Text>
            <Text style={styles.emptyTitle}>No orders yet</Text>
            <Text style={styles.emptySubtitle}>
              Browse chefs to place your first order!
            </Text>
          </View>
        }
        contentContainerStyle={
          allOrders.length === 0 ? styles.emptyContent : styles.listContent
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F9FAFB',
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 4,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#111827',
  },
  filterRow: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    gap: 8,
  },
  filterChip: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: '#F3F4F6',
    marginRight: 8,
  },
  filterChipActive: {
    backgroundColor: '#F97316',
  },
  filterChipText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#6B7280',
  },
  filterChipTextActive: {
    color: '#FFFFFF',
  },
  listContent: {
    paddingBottom: 24,
  },
  emptyContent: {
    flex: 1,
  },
  emptyState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: 80,
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#374151',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#9CA3AF',
    textAlign: 'center',
    paddingHorizontal: 40,
  },
  footerLoader: {
    paddingVertical: 16,
  },
});
