import { ScrollView, Text, View } from 'react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useAdminAnalytics } from '../../hooks/useAdminDashboard';
import {
  Card,
  ErrorState,
  LoadingState,
  ScreenHeader,
  SectionTitle,
  StatCard,
} from '../../components/kit';
import { formatINR, formatCount, titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;

export default function AnalyticsScreen() {
  const q = useAdminAnalytics();

  const byStatus = q.data?.ordersByStatus ?? {};
  const statusEntries = Object.entries(byStatus);
  const maxCount = statusEntries.reduce((m, [, v]) => Math.max(m, v), 0) || 1;

  return (
    <Screen>
      <ScreenHeader title="Analytics" back subtitle="Platform performance" />
      {q.isLoading ? (
        <LoadingState label="Loading analytics…" />
      ) : q.isError || !q.data ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
          <View style={{ flexDirection: 'row', gap: 10, marginBottom: 10 }}>
            <StatCard label="Total revenue" value={formatINR(q.data.overview.totalRevenue)} />
            <StatCard label="Total orders" value={formatCount(q.data.overview.totalOrders)} />
          </View>
          <View style={{ flexDirection: 'row', gap: 10 }}>
            <StatCard label="Avg order value" value={formatINR(q.data.overview.avgOrderValue)} />
            <StatCard label="Active users" value={formatCount(q.data.overview.activeUsers)} />
          </View>

          <SectionTitle>Orders by status</SectionTitle>
          {statusEntries.length === 0 ? (
            <Card>
              <Text style={{ fontFamily: 'Inter', fontSize: 14, color: c.ink.muted }}>No data.</Text>
            </Card>
          ) : (
            <Card style={{ gap: 12 }}>
              {statusEntries.map(([status, count]) => (
                <View key={status}>
                  <View style={{ flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text style={{ fontFamily: 'Inter-Medium', fontSize: 13, color: c.ink.DEFAULT }}>
                      {titleCase(status)}
                    </Text>
                    <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 13, color: c.ink.DEFAULT }}>
                      {formatCount(count)}
                    </Text>
                  </View>
                  <View style={{ height: 8, borderRadius: 4, backgroundColor: c.bone, overflow: 'hidden' }}>
                    <View
                      style={{
                        height: 8,
                        borderRadius: 4,
                        width: `${Math.max(4, (count / maxCount) * 100)}%`,
                        backgroundColor: c.ink.DEFAULT,
                      }}
                    />
                  </View>
                </View>
              ))}
            </Card>
          )}
        </ScrollView>
      )}
    </Screen>
  );
}
