import { useEffect, useMemo } from 'react';
import {
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import {
  useVendorDashboard,
  useToggleAcceptingOrders,
} from '../../hooks/useVendorDashboard';
import { useAuthStore } from '../../store/auth-store';
import { DashboardStatsCard } from '../../components/vendor/DashboardStatsCard';

/**
 * Pull a human-readable display name out of the user object. Priority:
 * explicit `name`, then the local-part of the email (Gmail handle), then
 * a generic "Chef" fallback. Strips any trailing service suffix from the
 * Gmail local part (e.g. "+vendor").
 */
function deriveDisplayName(
  user: { name?: string; email?: string } | null | undefined,
): string {
  if (user?.name && user.name.trim().length > 0) return user.name.trim();
  const email = user?.email ?? '';
  if (email.includes('@')) {
    const local = email.split('@')[0]?.split('+')[0] ?? '';
    if (local.length > 0) {
      // Title-case "first.last" → "First Last"
      return local
        .split(/[._-]/)
        .filter(Boolean)
        .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
        .join(' ');
    }
  }
  return 'Chef';
}

function StatusBadge({ status }: { status: string }) {
  const palette = STATUS_PALETTE[status] ?? STATUS_PALETTE.default;
  return (
    <View style={[styles.badge, { backgroundColor: palette.bg }]}>
      <Text style={[styles.badgeLabel, { color: palette.fg }]}>
        {status.replace(/_/g, ' ')}
      </Text>
    </View>
  );
}

const STATUS_PALETTE: Record<string, { bg: string; fg: string }> = {
  pending: { bg: theme.colors.amber.tint, fg: '#7A5A1A' },
  accepted: { bg: theme.colors.info.tint, fg: theme.colors.info.DEFAULT },
  preparing: { bg: theme.colors.herb.tint, fg: theme.colors.herb.soft },
  ready: { bg: '#E0E7FF', fg: '#3730A3' },
  delivered: { bg: theme.colors.herb.tint, fg: theme.colors.herb.soft },
  picked_up: { bg: theme.colors.herb.tint, fg: theme.colors.herb.soft },
  cancelled: { bg: theme.colors.destructive.tint, fg: theme.colors.destructive.DEFAULT },
  rejected: { bg: theme.colors.destructive.tint, fg: theme.colors.destructive.DEFAULT },
  default: { bg: theme.colors.mist.DEFAULT, fg: theme.colors.ink.soft },
};

export default function DashboardScreen() {
  const user = useAuthStore((s) => s.user);
  const {
    data: dashboard,
    isLoading,
    isRefetching,
    refetch,
    isError,
    error,
  } = useVendorDashboard();
  const toggleMutation = useToggleAcceptingOrders();

  const greeting = useMemo(() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  }, []);

  const displayName = deriveDisplayName(
    user as { name?: string; email?: string } | null,
  );

  // Compute "next action" — the one thing the chef should be doing right
  // now. Uber Driver does this: a single CTA at the top so the screen
  // answers "what do I do?" before showing raw numbers.
  const pendingCount = useMemo(
    () =>
      (dashboard?.recentOrders ?? []).filter((o) => o.status === 'pending')
        .length,
    [dashboard?.recentOrders],
  );

  const nextAction = useMemo(() => {
    if (pendingCount > 0) {
      return {
        kind: 'pending' as const,
        title: pendingCount === 1 ? '1 order awaiting acceptance' : `${pendingCount} orders awaiting acceptance`,
        cta: 'Open orders',
        onPress: () => router.push('/(tabs)/orders'),
      };
    }
    if (dashboard && !dashboard.acceptingOrders) {
      return {
        kind: 'closed' as const,
        title: 'Kitchen is closed',
        cta: 'Start accepting',
        onPress: () =>
          toggleMutation.mutate(true),
      };
    }
    return null;
  }, [pendingCount, dashboard, toggleMutation]);

  // Dashboard 404 with "Chef profile not found" means the user signed in
  // but never completed onboarding. Auto-route to the wizard.
  useEffect(() => {
    if (!isError) return;
    const status = (error as { response?: { data?: { error?: string } } } | null)
      ?.response?.data?.error;
    if (status === 'Chef profile not found') {
      router.replace('/(onboarding)/personal-info');
    }
  }, [isError, error]);

  if (isError) {
    return (
      <SafeAreaView style={styles.errorScreen}>
        <Text style={styles.errorTitle}>Couldn't load your dashboard</Text>
        <Text style={styles.errorBody}>
          Check your connection and try again.
        </Text>
        <Pressable onPress={() => refetch()} style={styles.errorPrimary}>
          <Text style={styles.errorPrimaryText}>Retry</Text>
        </Pressable>
        <Pressable
          onPress={() => router.replace('/(onboarding)/personal-info')}
          style={styles.errorSecondary}
        >
          <Text style={styles.errorSecondaryText}>Complete onboarding</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
      >
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.greeting}>{greeting}</Text>
          <Text style={styles.displayName}>{displayName}</Text>
        </View>

        {/* Next action — surfaces above stats, only when there's something to do */}
        {nextAction ? (
          <Pressable onPress={nextAction.onPress} style={styles.actionCard}>
            <View style={styles.actionLeft}>
              <Text style={styles.actionLabel}>NEXT</Text>
              <Text style={styles.actionTitle}>{nextAction.title}</Text>
            </View>
            <Text style={styles.actionCta}>{nextAction.cta} ›</Text>
          </Pressable>
        ) : null}

        {/* Stats grid 2×2 */}
        {isLoading ? (
          <View style={styles.statsGrid}>
            <View style={styles.statsRow}>
              <Skeleton height={92} style={{ flex: 1, borderRadius: theme.radius.md }} />
              <Skeleton height={92} style={{ flex: 1, borderRadius: theme.radius.md }} />
            </View>
            <View style={styles.statsRow}>
              <Skeleton height={92} style={{ flex: 1, borderRadius: theme.radius.md }} />
              <Skeleton height={92} style={{ flex: 1, borderRadius: theme.radius.md }} />
            </View>
          </View>
        ) : (
          <View style={styles.statsGrid}>
            <View style={styles.statsRow}>
              <DashboardStatsCard
                title="Today's orders"
                value={dashboard?.todayOrders ?? 0}
              />
              <DashboardStatsCard
                title="Today's earnings"
                value={`₹${(dashboard?.todayEarnings ?? 0).toFixed(0)}`}
              />
            </View>
            <View style={styles.statsRow}>
              <DashboardStatsCard
                title="Rating"
                value={(dashboard?.rating ?? 0).toFixed(1)}
                subtitle={`${dashboard?.totalReviews ?? 0} review${(dashboard?.totalReviews ?? 0) === 1 ? '' : 's'}`}
              />
              <DashboardStatsCard
                title="Reviews"
                value={dashboard?.totalReviews ?? 0}
                subtitle="total"
              />
            </View>
          </View>
        )}

        {/* Accepting Orders toggle */}
        <View style={styles.toggleCard}>
          <View style={{ flex: 1 }}>
            <Text style={styles.toggleTitle}>Accepting orders</Text>
            <Text style={styles.toggleSubtitle}>
              {dashboard?.acceptingOrders
                ? 'Customers can place orders'
                : 'Your kitchen is closed'}
            </Text>
          </View>
          <Switch
            value={dashboard?.acceptingOrders ?? false}
            onValueChange={(v) => toggleMutation.mutate(v)}
            trackColor={{
              false: theme.colors.mist.strong,
              true: theme.colors.ink.DEFAULT,
            }}
            thumbColor={theme.colors.paper}
            ios_backgroundColor={theme.colors.mist.strong}
            disabled={toggleMutation.isPending || isLoading}
          />
        </View>

        {/* Recent Orders */}
        {!isLoading && (dashboard?.recentOrders?.length ?? 0) > 0 && (
          <View style={styles.ordersBlock}>
            <View style={styles.ordersHeader}>
              <Text style={styles.ordersTitle}>Recent orders</Text>
              <Pressable
                onPress={() => router.push('/(tabs)/orders')}
                hitSlop={8}
              >
                <Text style={styles.ordersSeeAll}>See all</Text>
              </Pressable>
            </View>
            <View style={styles.ordersList}>
              {(dashboard?.recentOrders ?? []).slice(0, 3).map((order) => (
                <View key={order.id} style={styles.orderRow}>
                  <View style={{ flex: 1 }}>
                    <Text style={styles.orderName}>{order.customerName}</Text>
                    <Text style={styles.orderTotal}>
                      ₹{order.total.toFixed(0)}
                    </Text>
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

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[8],
  },

  header: {
    marginTop: theme.spacing[3],
    marginBottom: theme.spacing[5],
  },
  greeting: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  displayName: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    marginTop: 2,
  },

  actionCard: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[4],
  },
  actionLeft: { flex: 1, gap: 2 },
  actionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.2,
    color: theme.colors.herb.tint,
  },
  actionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  actionCta: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    marginLeft: theme.spacing[3],
  },

  statsGrid: { gap: theme.spacing[3], marginBottom: theme.spacing[4] },
  statsRow: { flexDirection: 'row', gap: theme.spacing[3] },

  toggleCard: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[5],
  },
  toggleTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  toggleSubtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },

  ordersBlock: { gap: theme.spacing[2] },
  ordersHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: theme.spacing[2],
  },
  ordersTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
  },
  ordersSeeAll: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
  ordersList: { gap: theme.spacing[2] },
  orderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: theme.radius.DEFAULT,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  orderName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  orderTotal: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
    fontVariant: ['tabular-nums'],
  },

  badge: {
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
  },
  badgeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    textTransform: 'capitalize',
    letterSpacing: 0.2,
  },

  errorScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.paper,
  },
  errorTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
    marginBottom: theme.spacing[2],
  },
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    marginBottom: theme.spacing[6],
  },
  errorPrimary: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorPrimaryText: {
    color: theme.colors.paper,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
  },
  errorSecondary: {
    marginTop: theme.spacing[3],
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorSecondaryText: {
    color: theme.colors.ink.DEFAULT,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    textDecorationLine: 'underline',
  },
});
