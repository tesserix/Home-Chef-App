import { useEffect } from 'react';
import { View, Text, ScrollView, Switch, RefreshControl, Pressable, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useVendorDashboard, useToggleAcceptingOrders } from '../../hooks/useVendorDashboard';
import { useAuthStore } from '../../store/auth-store';
import { DashboardStatsCard } from '../../components/vendor/DashboardStatsCard';

// Inline brand fallback — NativeWind className doesn't reach this error
// branch consistently when the dashboard query errors before tailwind
// styles hydrate. See task: Fix NativeWind v5 className not applying.
const ERR = StyleSheet.create({
  screen: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 24, backgroundColor: '#FBFAF7' },
  msg: { textAlign: 'center', fontSize: 16, color: '#B22B0E', marginBottom: 16 },
  ctaPrimary: { marginTop: 16, borderRadius: 12, backgroundColor: '#C2410C', paddingHorizontal: 24, paddingVertical: 12 },
  ctaPrimaryText: { color: '#FFFFFF', fontWeight: '600' },
  ctaSecondary: { marginTop: 12, borderRadius: 12, borderWidth: 1, borderColor: '#1A1A18', paddingHorizontal: 24, paddingVertical: 12 },
  ctaSecondaryText: { color: '#1A1A18', fontWeight: '600' },
});

function SkeletonBox({ className }: { className: string }) {
  return <View className={`animate-pulse rounded-2xl bg-mist ${className}`} />;
}

function StatusBadge({ status }: { status: string }) {
  const colorMap: Record<string, string> = {
    pending: 'bg-amber-tint text-amber',
    accepted: 'bg-info/10 text-info',
    preparing: 'bg-herb-tint text-herb',
    ready: 'bg-purple-100 text-info',
    delivered: 'bg-herb-tint text-herb',
    cancelled: 'bg-paprika-tint text-paprika',
    rejected: 'bg-paprika-tint text-paprika',
    picked_up: 'bg-herb-tint text-herb',
  };
  const colorClass = colorMap[status] ?? 'bg-mist text-ink-soft';
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
  const { data: dashboard, isLoading, isRefetching, refetch, isError, error } = useVendorDashboard();
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

  // Dashboard 404 with "Chef profile not found" means the user signed in but
  // never completed onboarding. Auto-redirect to the onboarding wizard
  // instead of stranding them on a Retry button that will never succeed.
  useEffect(() => {
    if (!isError) return;
    const status = (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
    if (status === 'Chef profile not found') {
      router.replace('/(onboarding)/personal-info');
    }
  }, [isError, error]);

  if (isError) {
    return (
      <SafeAreaView style={ERR.screen}>
        <Text style={ERR.msg}>Failed to load dashboard</Text>
        <Pressable onPress={() => refetch()} style={ERR.ctaPrimary}>
          <Text style={ERR.ctaPrimaryText}>Retry</Text>
        </Pressable>
        <Pressable onPress={() => router.replace('/(onboarding)/personal-info')} style={ERR.ctaSecondary}>
          <Text style={ERR.ctaSecondaryText}>Complete onboarding</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <ScrollView
        className="flex-1"
        contentContainerClassName="px-4 pb-8"
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#C2410C" />
        }
      >
        {/* Header */}
        <View className="mb-6 mt-4">
          <Text className="font-display text-2xl font-semibold text-ink">
            {greeting}, {displayName}
          </Text>
          <Text className="mt-1 text-sm text-ink-muted">Here's your kitchen overview</Text>
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
        <View className="mb-4 flex-row items-center justify-between rounded-2xl bg-bone px-4 py-3 shadow-sm border border-mist">
          <View>
            <Text className="text-base font-semibold text-ink">Accepting Orders</Text>
            <Text className="text-xs text-ink-muted mt-0.5">
              {dashboard?.acceptingOrders ? 'Customers can place orders' : 'Kitchen is closed'}
            </Text>
          </View>
          <Switch
            value={dashboard?.acceptingOrders ?? false}
            onValueChange={(v) => toggleMutation.mutate(v)}
            trackColor={{ false: '#d4d3ce', true: '#C2410C' }}
            thumbColor="#fafaf7"
            disabled={toggleMutation.isPending || isLoading}
          />
        </View>

        {/* Recent Orders */}
        {!isLoading && (dashboard?.recentOrders?.length ?? 0) > 0 && (
          <View>
            <Text className="mb-3 text-base font-semibold text-ink">Recent Orders</Text>
            <View className="gap-2">
              {(dashboard?.recentOrders ?? []).slice(0, 3).map((order) => (
                <View
                  key={order.id}
                  className="flex-row items-center justify-between rounded-xl bg-bone px-4 py-3 border border-mist"
                >
                  <View className="flex-1">
                    <Text className="text-sm font-medium text-ink">{order.customerName}</Text>
                    <Text className="text-xs text-ink-muted mt-0.5">₹{order.total.toFixed(0)}</Text>
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
