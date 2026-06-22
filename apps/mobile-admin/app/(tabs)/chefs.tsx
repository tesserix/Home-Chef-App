import { useState } from 'react';
import { FlatList, RefreshControl, View } from 'react-native';
import { router } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAdminChefs } from '../../hooks/useAdminChefs';
import type { ChefWithStats } from '../../lib/admin-types';
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
import { formatINR, errorMessage } from '../../lib/format';

const STATUS_OPTIONS = [
  { key: '', label: 'All' },
  { key: 'pending', label: 'Pending' },
  { key: 'verified', label: 'Verified' },
  { key: 'suspended', label: 'Suspended' },
] as const;

export function chefStatus(ch: ChefWithStats): { label: string; tone: Tone } {
  if (!ch.isVerified) return { label: 'Pending', tone: 'warning' };
  if (!ch.isActive) return { label: 'Suspended', tone: 'danger' };
  return { label: 'Verified', tone: 'success' };
}

export default function ChefsScreen() {
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<string>('');
  const query = useAdminChefs({ search, status });

  const chefs = query.data?.data ?? [];

  return (
    <Screen>
      <ScreenHeader
        title="Chefs"
        subtitle={
          query.data ? `${query.data.pagination.total} home kitchens` : 'Home kitchens'
        }
      />
      <View style={{ gap: 10, marginBottom: 4 }}>
        <SearchBar value={search} onChangeText={setSearch} placeholder="Search by name or owner…" />
        <FilterChips
          options={STATUS_OPTIONS.map((o) => ({ key: o.key, label: o.label }))}
          value={status}
          onChange={setStatus}
        />
      </View>

      {query.isLoading ? (
        <LoadingList />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : (
        <FlatList
          data={chefs}
          keyExtractor={(c) => c.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching}
              onRefresh={() => query.refetch()}
              tintColor="#0E0E0C"
            />
          }
          ListEmptyComponent={
            <Empty title="No chefs found" body="Try a different search or filter." />
          }
          renderItem={({ item }) => {
            const s = chefStatus(item);
            return (
              <ListItem
                title={item.businessName || 'Unnamed kitchen'}
                subtitle={item.ownerName || item.ownerEmail}
                meta={`${item.totalOrders} orders · ${item.rating?.toFixed(1) ?? '0.0'}★ · ${item.menuItemCount} items`}
                right={formatINR(item.totalRevenue)}
                badge={<Badge label={s.label} tone={s.tone} />}
                onPress={() => router.push({ pathname: '/chef/[id]', params: { id: item.id } })}
              />
            );
          }}
        />
      )}
    </Screen>
  );
}
