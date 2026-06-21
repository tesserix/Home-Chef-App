import { useState } from 'react';
import { FlatList, RefreshControl } from 'react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAdminMealPlans } from '../../hooks/useAdminMealPlans';
import {
  Badge,
  Empty,
  ErrorState,
  FilterChips,
  ListItem,
  LoadingList,
  ScreenHeader,
  type Tone,
} from '../../components/kit';
import { formatINR, formatDate, titleCase, errorMessage } from '../../lib/format';

const STATUS_OPTIONS = [
  { key: '', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'paused', label: 'Paused' },
  { key: 'cancelled', label: 'Cancelled' },
];

function tone(status: string): Tone {
  if (status === 'active') return 'success';
  if (status === 'cancelled') return 'danger';
  if (status === 'paused') return 'warning';
  return 'neutral';
}

export default function MealPlansScreen() {
  const [status, setStatus] = useState('');
  const q = useAdminMealPlans({ status });
  const plans = q.data?.data ?? [];

  return (
    <Screen>
      <ScreenHeader
        title="Meal Plans"
        back
        subtitle={q.data ? `${q.data.pagination.total} subscriptions` : 'Subscription oversight'}
      />
      <FilterChips options={STATUS_OPTIONS} value={status} onChange={setStatus} />

      {q.isLoading ? (
        <LoadingList />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <FlatList
          data={plans}
          keyExtractor={(p) => p.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={q.isRefetching} onRefresh={() => q.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No meal plans" />}
          renderItem={({ item }) => (
            <ListItem
              title={`Plan ${item.id.slice(0, 8)}`}
              subtitle={
                item.startDate
                  ? `${formatDate(item.startDate)} → ${item.endDate ? formatDate(item.endDate) : 'ongoing'}`
                  : `${item.mealCount ?? 0} meals`
              }
              meta={
                item.daysPerWeek
                  ? `${item.daysPerWeek} days/wk · ${item.mealCount ?? 0} meals`
                  : undefined
              }
              right={item.totalPrice != null ? formatINR(item.totalPrice) : undefined}
              badge={<Badge label={titleCase(item.status)} tone={tone(item.status)} />}
              chevron={false}
            />
          )}
        />
      )}
    </Screen>
  );
}
