import { useState } from 'react';
import { Alert, FlatList, RefreshControl, Text, View } from 'react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useAdminReviews,
  useHideReview,
  useUnhideReview,
} from '../../hooks/useAdminReviews';
import type { ReviewRow } from '../../lib/admin-types';
import {
  Badge,
  Card,
  Empty,
  ErrorState,
  FilterChips,
  LoadingList,
  ScreenHeader,
} from '../../components/kit';
import { PromptModal } from '../../components/PromptModal';
import { formatRelative, errorMessage } from '../../lib/format';

const c = theme.colors;
const VIEW_OPTIONS = [
  { key: 'visible', label: 'Visible' },
  { key: 'hidden', label: 'Hidden' },
];

export default function ReviewsScreen() {
  const [view, setView] = useState('visible');
  const q = useAdminReviews({ hidden: view === 'hidden' });
  const hide = useHideReview();
  const unhide = useUnhideReview();
  const [hideTarget, setHideTarget] = useState<ReviewRow | null>(null);
  const reviews = q.data?.data ?? [];

  const doUnhide = (r: ReviewRow) => {
    unhide.mutate(r.id, { onError: (e) => Alert.alert('Failed', errorMessage(e)) });
  };

  return (
    <Screen>
      <ScreenHeader
        title="Reviews"
        back
        subtitle={q.data ? `${q.data.pagination.total} ${view}` : 'Moderation'}
      />
      <FilterChips options={VIEW_OPTIONS} value={view} onChange={setView} />

      {q.isLoading ? (
        <LoadingList />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <FlatList
          data={reviews}
          keyExtractor={(r) => r.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24, gap: 10 }}
          refreshControl={
            <RefreshControl refreshing={q.isRefetching} onRefresh={() => q.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No reviews" />}
          renderItem={({ item }) => (
            <Card>
              <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <Badge label={`${item.rating?.toFixed(1) ?? '0.0'} ★`} tone={item.rating >= 4 ? 'success' : item.rating >= 3 ? 'warning' : 'danger'} />
                <Text style={{ fontFamily: 'Inter', fontSize: 12, color: c.ink.muted }}>
                  {formatRelative(item.createdAt)}
                </Text>
              </View>
              <Text style={{ fontFamily: 'Inter', fontSize: 14, color: c.ink.DEFAULT }}>
                {item.text || item.comment || 'No comment'}
              </Text>
              {item.isHidden && item.hiddenReason ? (
                <Text style={{ fontFamily: 'Inter', fontSize: 12, color: c.destructive.DEFAULT, marginTop: 6 }}>
                  Hidden: {item.hiddenReason}
                </Text>
              ) : null}
              <View style={{ marginTop: 10 }}>
                {item.isHidden ? (
                  <Text
                    onPress={() => doUnhide(item)}
                    style={{ fontFamily: 'Inter-SemiBold', fontSize: 14, color: c.success.soft }}
                  >
                    Unhide review
                  </Text>
                ) : (
                  <Text
                    onPress={() => setHideTarget(item)}
                    style={{ fontFamily: 'Inter-SemiBold', fontSize: 14, color: c.destructive.DEFAULT }}
                  >
                    Hide review
                  </Text>
                )}
              </View>
            </Card>
          )}
        />
      )}

      <PromptModal
        visible={!!hideTarget}
        title="Hide review"
        message="Reason for moderation (kept for audit)."
        placeholder="Reason…"
        confirmLabel="Hide"
        destructive
        submitting={hide.isPending}
        onClose={() => setHideTarget(null)}
        onConfirm={(reason) =>
          hideTarget &&
          hide.mutate(
            { id: hideTarget.id, reason },
            {
              onError: (e) => Alert.alert('Failed', errorMessage(e)),
              onSuccess: () => setHideTarget(null),
            }
          )
        }
      />
    </Screen>
  );
}
