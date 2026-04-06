import { View, Text, ScrollView, Switch, RefreshControl, Pressable } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useVendorDashboard, useToggleAcceptingOrders } from '../../hooks/useVendorDashboard';
import { useAuthStore } from '../../store/auth-store';
import { DashboardStatsCard } from '../../components/vendor/DashboardStatsCard';

function SkeletonBox({ className }: { className: string }) {
  return <View className={`animate-pulse rounded-2xl bg-gray-200 ${className}`} />;
}

function StatusBadge({ status }: { status: string }) {
  const colorMap: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-700',
    accepted: 'bg-blue-100 text-blue-700',
    preparing: 'bg-orange-100 text-orange-700',
    ready: 'bg-purple-100 text-purple-700',
    delivered: 'bg-green-100 text-green-700',
    cancelled: 'bg-red-100 text-red-700',
    rejected: 'bg-red-100 text-red-700',
    picked_up: 'bg-teal-100 text-teal-700',
  };
  const colorClass = colorMap[status] ?? 'bg-gray-100 text-gray-600';
  return (
    <View className={`rounded-full px-2 py-0.5 ${colorClass}`}>
      <Text className={`text-xs font-medium capitalize ${colorClass.split(' ')[1]}`}>
        {status.replace('_', ' ')}
      </Text>
    </View>
  );
}

export default function DashboardScreen() {
  const user = useAuthStore((s) => s.user);
  const { data: dashboard, isLoading, isRefetching, refetch, isError } = useVendorDashboard();
  const toggleMutation = useToggleAcceptingOrders();

  const greeting = (() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  })();

  const displayName = (user as { name?: string; email?: string } | null)?.name
    ?? (user as { email?: string } | null)?.email
    ?? 'Chef';

  if (isError) {
    return (
      <SafeAreaView className="flex-1 items-center justify-center bg-gray-50 px-6">
        <Text className="text-center text-base text-red-500">Failed to load dashboard</Text>
        <Pressable
          onPress={() => refetch()}
          className="mt-4 rounded-xl bg-orange-500 px-6 py-3 active:opacity-80"
        >
          <Text className="font-semibold text-white">Retry</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <ScrollView
        className="flex-1"
        contentContainerClassName="px-4 pb-8"
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#f97316" />
        }
      >
        {/* Header */}
        <View className="mb-6 mt-4">
          <Text className="text-2xl font-bold text-gray-900">
            {greeting}, {displayName}
          </Text>
          <Text className="mt-1 text-sm text-gray-500">Here's your kitchen overview</Text>
        </View>

        {/* Stats grid */}
        {isLoading ? (
          <View className="mb-4 flex-row gap-3">
            <View className="flex-1 gap-3">
              <SkeletonBox className="h-24" />
              <SkeletonBox className="h-24" />
            </View>
            <View className="flex-1 gap-3">
              <SkeletonBox className="h-24" />
              <SkeletonBox className="h-24" />
            </View>
          </View>
        ) : (
          <View className="mb-4 gap-3">
            <View className="flex-row gap-3">
              <DashboardStatsCard
                title="Today's Orders"
                value={dashboard?.todayOrders ?? 0}
              />
              <DashboardStatsCard
                title="Today's Earnings"
                value={`₹${(dashboard?.todayEarnings ?? 0).toFixed(0)}`}
              />
            </View>
            <View className="flex-row gap-3">
              <DashboardStatsCard
                title="Rating"
                value={`${(dashboard?.rating ?? 0).toFixed(1)} ⭐`}
              />
              <DashboardStatsCard
                title="Reviews"
                value={`${dashboard?.totalReviews ?? 0}`}
                subtitle="total reviews"
              />
            </View>
          </View>
        )}

        {/* Accepting Orders toggle */}
        <View className="mb-4 flex-row items-center justify-between rounded-2xl bg-white px-4 py-3 shadow-sm border border-gray-100">
          <View>
            <Text className="text-base font-semibold text-gray-900">Accepting Orders</Text>
            <Text className="text-xs text-gray-400 mt-0.5">
              {dashboard?.acceptingOrders ? 'Customers can place orders' : 'Kitchen is closed'}
            </Text>
          </View>
          <Switch
            value={dashboard?.acceptingOrders ?? false}
            onValueChange={(v) => toggleMutation.mutate(v)}
            trackColor={{ false: '#d1d5db', true: '#f97316' }}
            thumbColor="#ffffff"
            disabled={toggleMutation.isPending || isLoading}
          />
        </View>

        {/* Recent Orders */}
        {!isLoading && (dashboard?.recentOrders?.length ?? 0) > 0 && (
          <View>
            <Text className="mb-3 text-base font-semibold text-gray-900">Recent Orders</Text>
            <View className="gap-2">
              {(dashboard?.recentOrders ?? []).slice(0, 3).map((order) => (
                <View
                  key={order.id}
                  className="flex-row items-center justify-between rounded-xl bg-white px-4 py-3 border border-gray-100"
                >
                  <View className="flex-1">
                    <Text className="text-sm font-medium text-gray-900">{order.customerName}</Text>
                    <Text className="text-xs text-gray-400 mt-0.5">₹{order.total.toFixed(0)}</Text>
                  </View>
                  <StatusBadge status={order.status} />
                </View>
              ))}
            </View>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
