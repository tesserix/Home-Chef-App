import { useState } from 'react';
import { FlatList, RefreshControl, View } from 'react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import {
  useDeliveryStats,
  useDeliveryPartners,
  useDeliveryList,
} from '../../hooks/useAdminDelivery';
import {
  Badge,
  Empty,
  ErrorState,
  FilterChips,
  ListItem,
  LoadingList,
  ScreenHeader,
  StatCard,
  type Tone,
} from '../../components/kit';
import { formatINR, formatCount, titleCase, formatRelative, errorMessage } from '../../lib/format';
import { orderStatusTone } from '../(tabs)/orders';

const SEGMENTS = [
  { key: 'partners', label: 'Partners' },
  { key: 'deliveries', label: 'Deliveries' },
];

export default function DeliveryScreen() {
  const [seg, setSeg] = useState('partners');
  const stats = useDeliveryStats();
  const partners = useDeliveryPartners();
  const deliveries = useDeliveryList();
  const active = seg === 'partners' ? partners : deliveries;

  function partnerTone(verified: boolean, online: boolean): Tone {
    if (!verified) return 'warning';
    return online ? 'success' : 'neutral';
  }

  return (
    <Screen>
      <ScreenHeader
        title="Delivery"
        back
        subtitle={stats.data ? `${formatCount(stats.data.onlinePartners)} partners online` : 'Fleet & deliveries'}
      />

      {stats.data ? (
        <View style={{ gap: 10, marginBottom: 8 }}>
          <View style={{ flexDirection: 'row', gap: 10 }}>
            <StatCard label="Active deliveries" value={formatCount(stats.data.activeDeliveries)} />
            <StatCard label="Today" value={formatCount(stats.data.todayDeliveries)} />
          </View>
          <View style={{ flexDirection: 'row', gap: 10 }}>
            <StatCard label="Verified partners" value={formatCount(stats.data.verifiedPartners)} />
            <StatCard label="Today's payout" value={formatINR(stats.data.todayEarnings)} />
          </View>
        </View>
      ) : null}

      <FilterChips options={SEGMENTS} value={seg} onChange={setSeg} />

      {active.isLoading ? (
        <LoadingList />
      ) : active.isError ? (
        <ErrorState message={errorMessage(active.error)} onRetry={() => active.refetch()} />
      ) : seg === 'partners' ? (
        <FlatList
          data={partners.data?.data ?? []}
          keyExtractor={(p) => p.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={partners.isRefetching} onRefresh={() => partners.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No partners" />}
          renderItem={({ item }) => (
            <ListItem
              title={item.name || item.email}
              subtitle={`${titleCase(item.vehicleType || 'vehicle')} · ${item.totalDeliveries} deliveries`}
              meta={`${item.rating?.toFixed(1) ?? '0.0'}★`}
              badge={
                <Badge
                  label={!item.isVerified ? 'Unverified' : item.isOnline ? 'Online' : 'Offline'}
                  tone={partnerTone(item.isVerified, item.isOnline)}
                />
              }
              chevron={false}
            />
          )}
        />
      ) : (
        <FlatList
          data={deliveries.data?.data ?? []}
          keyExtractor={(d) => d.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={deliveries.isRefetching} onRefresh={() => deliveries.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No deliveries" />}
          renderItem={({ item }) => (
            <ListItem
              title={`#${item.orderId?.slice(0, 8) ?? item.id.slice(0, 8)}`}
              subtitle={
                item.pickup?.city || item.dropoff?.city
                  ? `${item.pickup?.city ?? '—'} → ${item.dropoff?.city ?? '—'}`
                  : undefined
              }
              meta={item.createdAt ? formatRelative(item.createdAt) : undefined}
              right={item.totalPayout != null ? formatINR(item.totalPayout) : undefined}
              badge={<Badge label={titleCase(item.status)} tone={orderStatusTone(item.status)} />}
              chevron={false}
            />
          )}
        />
      )}
    </Screen>
  );
}
