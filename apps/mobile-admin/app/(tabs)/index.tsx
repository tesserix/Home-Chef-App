import { useCallback } from 'react';
import { RefreshControl, ScrollView, View } from 'react-native';
import { router } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAuthStore } from '../../store/auth-store';
import { useAdminStats, useAdminActivities } from '../../hooks/useAdminDashboard';
import { useApprovalCounts } from '../../hooks/useAdminApprovals';
import {
  Card,
  ErrorState,
  ListItem,
  LoadingList,
  ScreenHeader,
  SectionTitle,
  StatCard,
  Badge,
} from '../../components/kit';
import { formatCount, formatINR, formatRelative, errorMessage } from '../../lib/format';

export default function DashboardScreen() {
  const user = useAuthStore((s) => s.user);
  const stats = useAdminStats();
  const activities = useAdminActivities(15);
  const counts = useApprovalCounts();

  const onRefresh = useCallback(() => {
    stats.refetch();
    activities.refetch();
    counts.refetch();
  }, [stats, activities, counts]);

  const pending = counts.data?.byStatus?.pending ?? stats.data?.pendingVerifications ?? 0;

  return (
    <Screen>
      <ScreenHeader
        title="Dashboard"
        subtitle={user?.email ? `Signed in as ${user.email}` : 'Home Chef platform overview'}
      />
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={{ paddingBottom: 32 }}
        refreshControl={
          <RefreshControl
            refreshing={stats.isRefetching || activities.isRefetching}
            onRefresh={onRefresh}
            tintColor="#0E0E0C"
          />
        }
      >
        {stats.isLoading ? (
          <LoadingList rows={4} />
        ) : stats.isError ? (
          <ErrorState message={errorMessage(stats.error)} onRetry={() => stats.refetch()} />
        ) : (
          <>
            {pending > 0 ? (
              <Card
                onPress={() => router.push('/approvals')}
                style={{ marginBottom: 12, flexDirection: 'row', alignItems: 'center', gap: 12 }}
              >
                <Badge label={`${pending} pending`} tone="warning" />
                <View style={{ flex: 1 }} />
                <Badge label="Review approvals →" tone="neutral" />
              </Card>
            ) : null}

            <View style={{ flexDirection: 'row', gap: 12, marginBottom: 12 }}>
              <StatCard
                label="Revenue"
                value={formatINR(stats.data?.revenue)}
                delta={
                  stats.data?.revenueChange != null
                    ? `${stats.data.revenueChange >= 0 ? '+' : ''}${stats.data.revenueChange.toFixed(1)}% vs yest.`
                    : undefined
                }
                deltaTone={(stats.data?.revenueChange ?? 0) >= 0 ? 'success' : 'danger'}
              />
              <StatCard
                label="Orders"
                value={formatCount(stats.data?.totalOrders)}
                delta={
                  stats.data?.ordersChange != null
                    ? `${stats.data.ordersChange >= 0 ? '+' : ''}${stats.data.ordersChange.toFixed(1)}% wk`
                    : undefined
                }
                deltaTone={(stats.data?.ordersChange ?? 0) >= 0 ? 'success' : 'danger'}
              />
            </View>
            <View style={{ flexDirection: 'row', gap: 12, marginBottom: 12 }}>
              <StatCard label="Today's orders" value={formatCount(stats.data?.ordersToday)} />
              <StatCard label="Today's revenue" value={formatINR(stats.data?.revenueToday)} />
            </View>
            <View style={{ flexDirection: 'row', gap: 12 }}>
              <StatCard label="Total chefs" value={formatCount(stats.data?.totalChefs)} />
              <StatCard
                label="Total users"
                value={formatCount(stats.data?.totalUsers)}
                delta={
                  stats.data?.newUsersToday
                    ? `+${formatCount(stats.data.newUsersToday)} today`
                    : undefined
                }
                deltaTone="success"
              />
            </View>

            <SectionTitle>Recent activity</SectionTitle>
            {activities.isLoading ? (
              <LoadingList rows={5} />
            ) : (activities.data?.length ?? 0) === 0 ? (
              <Card>
                <ListItem title="No recent activity" chevron={false} />
              </Card>
            ) : (
              <Card style={{ paddingVertical: 0 }}>
                {activities.data!.map((a) => (
                  <ListItem
                    key={a.id}
                    title={a.title}
                    subtitle={a.description}
                    meta={formatRelative(a.timestamp)}
                    chevron={false}
                  />
                ))}
              </Card>
            )}
          </>
        )}
      </ScrollView>
    </Screen>
  );
}
