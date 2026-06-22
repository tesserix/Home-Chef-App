import { useState } from 'react';
import { FlatList, RefreshControl, View } from 'react-native';
import { router } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAdminOrders } from '../../hooks/useAdminOrders';
import {
  Badge,
  Empty,
  ErrorState,
  FilterChips,
  ListItem,
  LoadingList,
  ScreenHeader,
  SearchBar,
  type Tone,
} from '../../components/kit';
import { formatINR, formatRelative, titleCase, errorMessage } from '../../lib/format';

const STATUS_OPTIONS = [
  { key: '', label: 'All' },
  { key: 'pending', label: 'Pending' },
  { key: 'preparing', label: 'Preparing' },
  { key: 'out_for_delivery', label: 'On the way' },
  { key: 'delivered', label: 'Delivered' },
  { key: 'cancelled', label: 'Cancelled' },
];

export function orderStatusTone(status: string): Tone {
  switch (status) {
    case 'delivered':
      return 'success';
    case 'cancelled':
    case 'rejected':
      return 'danger';
    case 'pending':
      return 'warning';
    default:
      return 'info';
  }
}

export default function OrdersScreen() {
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const query = useAdminOrders({ search, status });
  const orders = query.data?.data ?? [];

  return (
    <Screen>
      <ScreenHeader
        title="Orders"
        subtitle={query.data ? `${query.data.pagination.total} total orders` : 'All orders'}
      />
      <View style={{ gap: 10, marginBottom: 4 }}>
        <SearchBar value={search} onChangeText={setSearch} placeholder="Search order # or customer…" />
        <FilterChips options={STATUS_OPTIONS} value={status} onChange={setStatus} />
      </View>

      {query.isLoading ? (
        <LoadingList />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : (
        <FlatList
          data={orders}
          keyExtractor={(o) => o.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching}
              onRefresh={() => query.refetch()}
              tintColor="#0E0E0C"
            />
          }
          ListEmptyComponent={<Empty title="No orders found" />}
          renderItem={({ item }) => (
            <ListItem
              title={`#${item.orderNumber || item.id.slice(0, 8)}`}
              subtitle={`${item.customerName} → ${item.chefName}`}
              meta={`${item.itemCount} items · ${formatRelative(item.createdAt)}`}
              right={formatINR(item.total)}
              badge={<Badge label={titleCase(item.status)} tone={orderStatusTone(item.status)} />}
              onPress={() => router.push({ pathname: '/order/[id]', params: { id: item.id } })}
            />
          )}
        />
      )}
    </Screen>
  );
}
