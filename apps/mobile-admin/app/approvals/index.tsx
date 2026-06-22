import { useState } from 'react';
import { FlatList, RefreshControl } from 'react-native';
import { router } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAdminApprovals } from '../../hooks/useAdminApprovals';
import type { ApprovalPriority, ApprovalStatus } from '../../lib/admin-types';
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
import { formatRelative, titleCase, errorMessage } from '../../lib/format';

const STATUS_OPTIONS = [
  { key: 'pending', label: 'Pending' },
  { key: 'info_requested', label: 'Info requested' },
  { key: 'approved', label: 'Approved' },
  { key: 'rejected', label: 'Rejected' },
];

function priorityTone(p: ApprovalPriority): Tone {
  switch (p) {
    case 'urgent':
      return 'danger';
    case 'high':
      return 'warning';
    case 'low':
      return 'neutral';
    default:
      return 'info';
  }
}

export default function ApprovalsScreen() {
  const [status, setStatus] = useState<ApprovalStatus>('pending');
  const q = useAdminApprovals({ status });
  const items = q.data?.data ?? [];

  return (
    <Screen>
      <ScreenHeader
        title="Approvals"
        back
        subtitle={q.data ? `${q.data.pagination.total} ${titleCase(status)}` : 'Review queue'}
      />
      <FilterChips
        options={STATUS_OPTIONS}
        value={status}
        onChange={(v) => setStatus(v as ApprovalStatus)}
      />

      {q.isLoading ? (
        <LoadingList />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(a) => a.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={q.isRefetching} onRefresh={() => q.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="Nothing here" body="No requests in this state." />}
          renderItem={({ item }) => (
            <ListItem
              title={item.title || titleCase(item.type)}
              subtitle={titleCase(item.type)}
              meta={formatRelative(item.createdAt)}
              badge={<Badge label={titleCase(item.priority)} tone={priorityTone(item.priority)} />}
              onPress={() => router.push({ pathname: '/approvals/[id]', params: { id: item.id } })}
            />
          )}
        />
      )}
    </Screen>
  );
}
